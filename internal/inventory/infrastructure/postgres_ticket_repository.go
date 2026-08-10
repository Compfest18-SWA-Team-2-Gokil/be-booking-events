package infrastructure

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/inventory/application"
	"github.com/ebk-tech/be-booking-events/internal/inventory/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTicketRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTicketRepository(pool *pgxpool.Pool) *PostgresTicketRepository {
	return &PostgresTicketRepository{pool: pool}
}

// Compile-time check: PostgresTicketRepository harus memenuhi kontrak ports.go.
var _ application.TicketUnitRepository = (*PostgresTicketRepository)(nil)

// HoldAvailableUnits adalah implementasi konkret dari anti-oversell engine.
//
// Strategi locking (TDD Bab 3): Pessimistic lock via SELECT FOR UPDATE.
// Strategi deadlock prevention (TDD Open Question #2 & #4):
//   1. Sort requests by TicketTypeID ASC → semua transaksi akuisisi lock tipe dalam urutan sama.
//   2. ORDER BY id ASC dalam setiap tipe → konsisten jika ada transaksi lain yang lock subset baris sama.
//
// Lazy expiry (TDD Bab 4): unit HELD yang kadaluarsa dirilis di dalam transaksi yang sama,
// sebelum SELECT FOR UPDATE — sehingga slot yang baru expired langsung tersedia untuk pembeli ini.
func (r *PostgresTicketRepository) HoldAvailableUnits(
	ctx context.Context,
	requests []application.HoldRequest,
	holdUntil time.Time,
) ([]*domain.TicketUnit, error) {
	// 1. Sort by TicketTypeID untuk jaminan urutan akuisisi lock yang konsisten.
	sorted := make([]application.HoldRequest, len(requests))
	copy(sorted, requests)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TicketTypeID < sorted[j].TicketTypeID
	})

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	typeIDs := make([]string, len(sorted))
	for i, req := range sorted {
		typeIDs[i] = req.TicketTypeID
	}

	// 2. Lazy expiry: rilis unit HELD kadaluarsa untuk semua tipe yang relevan.
	_, err = tx.Exec(ctx, `
		UPDATE ticket_units
		SET status = 'AVAILABLE', held_until = NULL, updated_at = NOW()
		WHERE ticket_type_id = ANY($1)
		  AND status = 'HELD'
		  AND held_until < NOW()
	`, typeIDs)
	if err != nil {
		return nil, fmt.Errorf("lazy expiry: %w", err)
	}

	var allUnits []*domain.TicketUnit

	// 3. Untuk setiap tipe (sudah diurutkan), SELECT FOR UPDATE.
	for _, req := range sorted {
		rows, err := tx.Query(ctx, `
			SELECT id, ticket_type_id, seat_label, status, held_until, order_id
			FROM ticket_units
			WHERE ticket_type_id = $1
			  AND status = 'AVAILABLE'
			ORDER BY id
			LIMIT $2
			FOR UPDATE
		`, req.TicketTypeID, req.Quantity)
		if err != nil {
			return nil, fmt.Errorf("select for update (type %s): %w", req.TicketTypeID, err)
		}

		units, err := pgx.CollectRows(rows, scanTicketUnit)
		if err != nil {
			return nil, fmt.Errorf("scan units: %w", err)
		}

		if len(units) < req.Quantity {
			// All-or-Nothing: kalau satu tipe tidak cukup, seluruh transaksi gagal.
			return nil, domain.ErrTicketNotAvailable
		}

		allUnits = append(allUnits, units...)
	}

	// 4. Update semua unit ke HELD dalam satu query.
	allIDs := make([]string, len(allUnits))
	for i, u := range allUnits {
		allIDs[i] = u.ID
	}

	_, err = tx.Exec(ctx, `
		UPDATE ticket_units
		SET status = 'HELD', held_until = $1, updated_at = NOW()
		WHERE id = ANY($2)
	`, holdUntil, allIDs)
	if err != nil {
		return nil, fmt.Errorf("update to held: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Sync in-memory state supaya caller tidak perlu re-fetch.
	for _, u := range allUnits {
		u.Status = domain.StatusHeld
		u.HeldUntil = &holdUntil
	}

	return allUnits, nil
}

// ExpireHeldUnits dipakai oleh cleanup worker (SKIP LOCKED agar tidak block transaksi aktif).
func (r *PostgresTicketRepository) ExpireHeldUnits(ctx context.Context) (int, error) {
	// SKIP LOCKED: baris yang sedang di-FOR UPDATE oleh transaksi lain dilewati —
	// worker tidak perlu menunggu dan tidak akan menghambat flow checkout.
	tag, err := r.pool.Exec(ctx, `
		UPDATE ticket_units
		SET status = 'AVAILABLE', held_until = NULL, updated_at = NOW()
		WHERE id IN (
			SELECT id FROM ticket_units
			WHERE status = 'HELD'
			  AND held_until < NOW()
			FOR UPDATE SKIP LOCKED
		)
	`)
	if err != nil {
		return 0, fmt.Errorf("expire held units: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresTicketRepository) UpdateStatus(
	ctx context.Context,
	unitID string,
	newStatus domain.TicketUnitStatus,
	orderID *string,
) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE ticket_units
		SET status = $1, order_id = $2, updated_at = NOW()
		WHERE id = $3
	`, string(newStatus), orderID, unitID)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func scanTicketUnit(row pgx.CollectableRow) (*domain.TicketUnit, error) {
	u := &domain.TicketUnit{}
	var status string
	err := row.Scan(&u.ID, &u.TicketTypeID, &u.SeatLabel, &status, &u.HeldUntil, &u.OrderID)
	if err != nil {
		return nil, err
	}
	u.Status = domain.TicketUnitStatus(status)
	return u, nil
}

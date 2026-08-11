package infrastructure

import (
	"context"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/checkin/application"
	"github.com/ebk-tech/be-booking-events/internal/checkin/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCheckinRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCheckinRepository(pool *pgxpool.Pool) *PostgresCheckinRepository {
	return &PostgresCheckinRepository{pool: pool}
}

var _ application.CheckinRepository = (*PostgresCheckinRepository)(nil)

// GetConfirmedUnit membaca tiket dari tabel milik modul Inventory (read-only cross-module).
func (r *PostgresCheckinRepository) GetConfirmedUnit(ctx context.Context, ticketUnitID string) (*domain.CheckinTicket, error) {
	var ticket domain.CheckinTicket
	var orderID *string

	err := r.pool.QueryRow(ctx, `
		SELECT tu.id, tu.order_id, tt.event_id::text
		FROM ticket_units tu
		JOIN ticket_types tt ON tt.id = tu.ticket_type_id
		WHERE tu.id = $1 AND tu.status = 'CONFIRMED'
	`, ticketUnitID).Scan(&ticket.ID, &orderID, &ticket.EventID)

	if err != nil {
		return nil, domain.ErrTicketNotConfirmed
	}

	if orderID != nil {
		ticket.OrderID = *orderID
	}

	return &ticket, nil
}

// AdmitUnit adalah atomic conditional UPDATE: hanya berhasil jika status masih CONFIRMED.
// Jika RETURNING kosong (0 baris), berarti sudah ADMITTED atau sudah REFUNDED.
func (r *PostgresCheckinRepository) AdmitUnit(ctx context.Context, ticketUnitID, gateOperatorID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE ticket_units
		SET status = 'ADMITTED',
		    admitted_at = NOW(),
		    admitted_by = $2,
		    updated_at = NOW()
		WHERE id = $1 AND status = 'CONFIRMED'
	`, ticketUnitID, gateOperatorID)

	if err != nil {
		return fmt.Errorf("admit unit: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrAlreadyAdmitted
	}

	return nil
}

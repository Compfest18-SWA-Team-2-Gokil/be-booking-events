package infrastructure

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/admin/application"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdminRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAdminRepository(pool *pgxpool.Pool) *PostgresAdminRepository {
	return &PostgresAdminRepository{pool: pool}
}

var _ application.AdminRepository = (*PostgresAdminRepository)(nil)

func (r *PostgresAdminRepository) ListDisputes(ctx context.Context, limit, offset int) ([]application.DisputeOrder, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM orders o
		WHERE o.status IN ('PAYMENT_DISCREPANCY', 'REFUND_REQUESTED', 'REFUND_ORGANIZER_APPROVED')
	`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count disputes: %w", err)
	}

	if limit <= 0 {
		limit = 10
	}

	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.buyer_id, u.email, o.event_id, o.status, o.total_amount, o.created_at, o.updated_at
		FROM orders o
		JOIN users u ON u.id = o.buyer_id
		WHERE o.status IN ('PAYMENT_DISCREPANCY', 'REFUND_REQUESTED', 'REFUND_ORGANIZER_APPROVED')
		ORDER BY o.updated_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list disputes: %w", err)
	}
	defer rows.Close()

	var result []application.DisputeOrder
	for rows.Next() {
		var d application.DisputeOrder
		if err := rows.Scan(&d.OrderID, &d.BuyerID, &d.BuyerEmail, &d.EventID,
			&d.Status, &d.TotalAmount, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		result = append(result, d)
	}
	return result, total, rows.Err()
}

func (r *PostgresAdminRepository) OverrideOrderStatus(ctx context.Context, orderID, adminID, newStatus, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&oldStatus); err != nil {
		return fmt.Errorf("order tidak ditemukan: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2
	`, newStatus, orderID); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if newStatus == "REFUNDED" {
		_, _ = tx.Exec(ctx, `
			UPDATE ticket_units
			SET status = 'AVAILABLE', order_id = NULL, updated_at = NOW()
			WHERE order_id = $1 AND status IN ('HELD', 'CONFIRMED', 'PAYMENT_PENDING')
		`, orderID)
	}

	// Audit log wajib — override admin harus traceable.
	meta := fmt.Sprintf(`{"reason":%q}`, reason)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, entity_name, entity_id, action, from_state, to_state, metadata)
		VALUES ($1, 'ADMIN', 'order', $2, 'ADMIN_OVERRIDE', $3, $4, $5)
	`, adminID, orderID, oldStatus, newStatus, meta); err != nil {
		return fmt.Errorf("audit log: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresAdminRepository) ReassignTicket(ctx context.Context, unitID, adminID, targetOrderID, newSeatNumber, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Pastikan ticket_unit ada
	var oldOrderID *string
	var oldSeatNumber *string
	var status string
	err = tx.QueryRow(ctx, `SELECT order_id, seat_number, status FROM ticket_units WHERE id = $1`, unitID).
		Scan(&oldOrderID, &oldSeatNumber, &status)
	if err != nil {
		return fmt.Errorf("unit tiket tidak ditemukan: %w", err)
	}

	// Pastikan target order ada
	var targetStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, targetOrderID).Scan(&targetStatus); err != nil {
		return fmt.Errorf("order tujuan tidak ditemukan: %w", err)
	}

	// Update ticket_unit ke order baru
	if newSeatNumber != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_units 
			SET order_id = $1, seat_number = $2, updated_at = NOW() 
			WHERE id = $3
		`, targetOrderID, newSeatNumber, unitID); err != nil {
			return fmt.Errorf("update ticket unit: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_units 
			SET order_id = $1, updated_at = NOW() 
			WHERE id = $2
		`, targetOrderID, unitID); err != nil {
			return fmt.Errorf("update ticket unit: %w", err)
		}
	}

	// Audit log wajib — reassign admin harus traceable
	oldOrderStr := ""
	if oldOrderID != nil {
		oldOrderStr = *oldOrderID
	}
	meta := fmt.Sprintf(`{"reason":%q,"from_order_id":%q,"to_order_id":%q,"new_seat_number":%q}`, reason, oldOrderStr, targetOrderID, newSeatNumber)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, entity_name, entity_id, action, from_state, to_state, metadata)
		VALUES ($1, 'ADMIN', 'ticket_unit', $2, 'ADMIN_REASSIGN', $3, $4, $5)
	`, adminID, unitID, oldOrderStr, targetOrderID, meta); err != nil {
		return fmt.Errorf("audit log: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresAdminRepository) ListAuditLogs(ctx context.Context, limit, offset int) ([]application.AuditLogEntry, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	rows, err := r.pool.Query(ctx, `
		SELECT 
			a.id::text, 
			COALESCE(a.actor_id::text, ''), 
			COALESCE(u.email, 'system'), 
			a.actor_role, 
			a.entity_name, 
			a.entity_id::text, 
			a.action, 
			a.from_state, 
			a.to_state, 
			COALESCE(a.metadata->>'reason', ''), 
			a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		ORDER BY a.created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var result []application.AuditLogEntry
	for rows.Next() {
		var entry application.AuditLogEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.ActorID,
			&entry.ActorEmail,
			&entry.ActorRole,
			&entry.EntityType,
			&entry.EntityID,
			&entry.Action,
			&entry.FromStatus,
			&entry.ToStatus,
			&entry.Reason,
			&entry.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, entry)
	}
	return result, total, rows.Err()
}

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/admin/application"
	auditdomain "github.com/ebk-tech/be-booking-events/internal/audit/domain"
	ordersdomain "github.com/ebk-tech/be-booking-events/internal/orders/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdminRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAdminRepository(pool *pgxpool.Pool) *PostgresAdminRepository {
	return &PostgresAdminRepository{pool: pool}
}

var _ application.AdminRepository = (*PostgresAdminRepository)(nil)

func (r *PostgresAdminRepository) ListDisputedOrders(ctx context.Context) ([]*ordersdomain.Order, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, buyer_id, event_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE status IN ('PAYMENT_DISCREPANCY', 'REFUND_REQUESTED')
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list disputes: %w", err)
	}
	defer rows.Close()

	var orders []*ordersdomain.Order
	for rows.Next() {
		o := &ordersdomain.Order{}
		if err := rows.Scan(&o.ID, &o.BuyerID, &o.EventID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *PostgresAdminRepository) OverrideOrderStatus(
	ctx context.Context,
	orderID string,
	newStatus ordersdomain.OrderStatus,
	reason string,
	actorID string,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&oldStatus)
	if err == pgx.ErrNoRows {
		return ordersdomain.ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("get current order: %w", err)
	}

	// Update orders table
	_, err = tx.Exec(ctx, `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`, string(newStatus), orderID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// Synchronize unit states if needed
	switch newStatus {
	case ordersdomain.OrderStatusPaid:
		_, err = tx.Exec(ctx, `UPDATE ticket_units SET status = 'CONFIRMED' WHERE order_id = $1`, orderID)
	case ordersdomain.OrderStatusRefunded:
		_, err = tx.Exec(ctx, `UPDATE ticket_units SET status = 'REFUNDED' WHERE order_id = $1`, orderID)
	case ordersdomain.OrderStatusCancelled:
		_, err = tx.Exec(ctx, `UPDATE ticket_units SET status = 'AVAILABLE', order_id = NULL WHERE order_id = $1`, orderID)
	}
	if err != nil {
		return fmt.Errorf("sync units: %w", err)
	}

	// Write immutable audit log
	metaJSON, _ := json.Marshal(map[string]any{"reason": reason})
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, entity_name, entity_id, action, from_state, to_state, metadata)
		VALUES ($1, 'ADMIN', 'orders', $2, 'ADMIN_OVERRIDE', $3, $4, $5)
	`, actorID, orderID, oldStatus, string(newStatus), metaJSON)
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresAdminRepository) ReassignTicket(
	ctx context.Context,
	unitID string,
	targetOrderID string,
	newSeatNumber string,
	reason string,
	actorID string,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldOrderID *string
	var oldSeat *string
	err = tx.QueryRow(ctx, `SELECT order_id, seat_number FROM ticket_units WHERE id = $1 FOR UPDATE`, unitID).Scan(&oldOrderID, &oldSeat)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("ticket unit not found")
	}
	if err != nil {
		return fmt.Errorf("query ticket unit: %w", err)
	}

	var seatVal any
	if newSeatNumber != "" {
		seatVal = newSeatNumber
	} else {
		seatVal = oldSeat
	}

	_, err = tx.Exec(ctx, `
		UPDATE ticket_units
		SET order_id = $1, seat_number = $2
		WHERE id = $3
	`, targetOrderID, seatVal, unitID)
	if err != nil {
		return fmt.Errorf("reassign ticket: %w", err)
	}

	fromMeta := map[string]any{"old_order_id": oldOrderID, "old_seat": oldSeat}
	toMeta := map[string]any{"target_order_id": targetOrderID, "new_seat": newSeatNumber, "reason": reason}
	metaJSON, _ := json.Marshal(map[string]any{"from": fromMeta, "to": toMeta})

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, entity_name, entity_id, action, metadata)
		VALUES ($1, 'ADMIN', 'ticket_units', $2, 'TICKET_REASSIGNMENT', $3)
	`, actorID, unitID, metaJSON)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresAdminRepository) ListAuditLogs(ctx context.Context, limit int) ([]*auditdomain.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_id, actor_role, entity_name, entity_id, action, from_state, to_state, metadata, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*auditdomain.AuditLog
	for rows.Next() {
		l := &auditdomain.AuditLog{}
		var actorRole, fromState, toState *string
		var metaRaw []byte
		if err := rows.Scan(&l.ID, &l.ActorID, &actorRole, &l.EntityName, &l.EntityID, &l.Action, &fromState, &toState, &metaRaw, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan log: %w", err)
		}
		if actorRole != nil {
			l.ActorRole = *actorRole
		}
		if fromState != nil {
			l.FromState = *fromState
		}
		if toState != nil {
			l.ToState = *toState
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &l.Metadata)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresOrderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOrderRepository(pool *pgxpool.Pool) *PostgresOrderRepository {
	return &PostgresOrderRepository{pool: pool}
}

var _ application.OrderRepository = (*PostgresOrderRepository)(nil)

// CreateOrder membuat order dan mengaitkan unit_ids ke order dalam satu transaksi.
// Total amount dihitung dari harga ticket_type masing-masing unit.
func (r *PostgresOrderRepository) CreateOrder(ctx context.Context, buyerID, eventID string, unitIDs []string) (*domain.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Pastikan semua unit HELD dan ambil total harga.
	var totalAmount int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(tt.price), 0)
		FROM ticket_units tu
		JOIN ticket_types tt ON tt.id = tu.ticket_type_id
		WHERE tu.id = ANY($1) AND tu.status = 'HELD' AND tu.order_id IS NULL
	`, unitIDs).Scan(&totalAmount)
	if err != nil {
		return nil, fmt.Errorf("hitung total: %w", err)
	}

	// Verifikasi semua unit valid (jumlah baris = jumlah unit_ids).
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM ticket_units
		WHERE id = ANY($1) AND status = 'HELD' AND order_id IS NULL
	`, unitIDs).Scan(&count); err != nil || count != len(unitIDs) {
		return nil, domain.ErrNoHeldUnits
	}

	order := &domain.Order{}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (buyer_id, event_id, status, total_amount)
		VALUES ($1, $2, 'PENDING', $3)
		RETURNING id, buyer_id, event_id, status, total_amount, created_at, updated_at
	`, buyerID, eventID, totalAmount).Scan(
		&order.ID, &order.BuyerID, &order.EventID, &order.Status,
		&order.TotalAmount, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	// Kaitkan unit ke order.
	_, err = tx.Exec(ctx, `
		UPDATE ticket_units SET order_id = $1
		WHERE id = ANY($2)
	`, order.ID, unitIDs)
	if err != nil {
		return nil, fmt.Errorf("link units: %w", err)
	}

	order.UnitIDs = unitIDs
	return order, tx.Commit(ctx)
}

func (r *PostgresOrderRepository) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	o := &domain.Order{}
	err := r.pool.QueryRow(ctx, `
		SELECT 
			o.id, 
			o.buyer_id, 
			o.event_id, 
			COALESCE(e.name, '') as event_name,
			o.status, 
			o.total_amount, 
			COALESCE(array_agg(tu.id::text) FILTER (WHERE tu.id IS NOT NULL), '{}') as unit_ids,
			o.created_at, 
			o.updated_at
		FROM orders o
		LEFT JOIN events e ON e.id = o.event_id
		LEFT JOIN ticket_units tu ON tu.order_id = o.id
		WHERE o.id = $1
		GROUP BY o.id, e.name
	`, orderID).Scan(
		&o.ID, &o.BuyerID, &o.EventID, &o.EventName, &o.Status,
		&o.TotalAmount, &o.UnitIDs, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

// GetOrdersByBuyer mengembalikan semua order milik buyer, diurutkan paling baru beserta unit_ids dan nama event.
func (r *PostgresOrderRepository) GetOrdersByBuyer(ctx context.Context, buyerID string) ([]*domain.Order, error) {
	// Auto-cancel order PENDING / PAYMENT_PENDING yang sudah lewat batas waktu (>= 10 menit atau yang tidak memiliki unit HELD lagi)
	_, _ = r.pool.Exec(ctx, `
		UPDATE orders
		SET status = 'CANCELLED', updated_at = NOW()
		WHERE buyer_id = $1
		  AND status IN ('PENDING', 'PAYMENT_PENDING')
		  AND (
		      created_at < NOW() - INTERVAL '10 minutes'
		      OR NOT EXISTS (
		          SELECT 1 FROM ticket_units WHERE order_id = orders.id AND status = 'HELD'
		      )
		  )
	`, buyerID)

	rows, err := r.pool.Query(ctx, `
		SELECT 
			o.id, 
			o.buyer_id, 
			o.event_id, 
			COALESCE(e.name, '') as event_name,
			o.status, 
			o.total_amount, 
			COALESCE(array_agg(tu.id::text) FILTER (WHERE tu.id IS NOT NULL), '{}') as unit_ids,
			o.created_at, 
			o.updated_at
		FROM orders o
		LEFT JOIN events e ON e.id = o.event_id
		LEFT JOIN ticket_units tu ON tu.order_id = o.id
		WHERE o.buyer_id = $1
		GROUP BY o.id, e.name
		ORDER BY o.created_at DESC
	`, buyerID)
	if err != nil {
		return nil, fmt.Errorf("get orders by buyer: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(
			&o.ID, &o.BuyerID, &o.EventID, &o.EventName, &o.Status,
			&o.TotalAmount, &o.UnitIDs, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// ConfirmOrderPayment atomik: update ticket_units HELD→CONFIRMED dan order→PAID.
// Jika 0 unit ter-update (tiket sudah direbut), order diset PAYMENT_DISCREPANCY dan return ErrLostSeat.
func (r *PostgresOrderRepository) ConfirmOrderPayment(ctx context.Context, orderID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE ticket_units SET status = 'CONFIRMED', updated_at = NOW()
		WHERE order_id = $1 AND status = 'HELD'
	`, orderID)
	if err != nil {
		return fmt.Errorf("confirm units: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// Tiket sudah direbut — tandai sebagai discrepancy, jangan set PAID.
		_, err = tx.Exec(ctx, `
			UPDATE orders SET status = 'PAYMENT_DISCREPANCY', updated_at = NOW() WHERE id = $1
		`, orderID)
		if err != nil {
			return fmt.Errorf("set discrepancy: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return domain.ErrLostSeat
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders SET status = 'PAID', updated_at = NOW() WHERE id = $1
	`, orderID)
	if err != nil {
		return fmt.Errorf("set paid: %w", err)
	}

	return tx.Commit(ctx)
}

// HasAdmittedUnits mengembalikan true jika ada tiket dalam order yang sudah di-scan gerbang.
func (r *PostgresOrderRepository) HasAdmittedUnits(ctx context.Context, orderID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM ticket_units WHERE order_id = $1 AND status = 'ADMITTED'
		)
	`, orderID).Scan(&exists)
	return exists, err
}

// UpdateOrderStatus juga update ticket_units status jika order berpindah ke PAID atau REFUNDED.
func (r *PostgresOrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2
	`, string(status), orderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	switch status {
	case domain.OrderStatusPaid:
		_, err = tx.Exec(ctx, `
			UPDATE ticket_units SET status = 'CONFIRMED'
			WHERE order_id = $1 AND status = 'HELD'
		`, orderID)
	case domain.OrderStatusRefunded:
		_, err = tx.Exec(ctx, `
			UPDATE ticket_units SET status = 'REFUNDED'
			WHERE order_id = $1
		`, orderID)
	case domain.OrderStatusCancelled:
		_, err = tx.Exec(ctx, `
			UPDATE ticket_units SET status = 'AVAILABLE', order_id = NULL
			WHERE order_id = $1 AND status = 'HELD'
		`, orderID)
	}
	if err != nil {
		return fmt.Errorf("update ticket units: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresOrderRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payments (order_id, amount, status, xendit_invoice_id, xendit_invoice_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, p.OrderID, p.Amount, string(p.Status), p.XenditInvoiceID, p.XenditInvoiceURL,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *PostgresOrderRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	p := &domain.Payment{}
	var method, invoiceID, invoiceURL, refundID *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, order_id, amount, status, payment_method,
		       xendit_invoice_id, xendit_invoice_url, xendit_refund_id,
		       created_at, updated_at
		FROM payments WHERE order_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, orderID).Scan(
		&p.ID, &p.OrderID, &p.Amount, &p.Status,
		&method, &invoiceID, &invoiceURL, &refundID,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}
	if method != nil {
		p.PaymentMethod = *method
	}
	if invoiceID != nil {
		p.XenditInvoiceID = *invoiceID
	}
	if invoiceURL != nil {
		p.XenditInvoiceURL = *invoiceURL
	}
	if refundID != nil {
		p.XenditRefundID = *refundID
	}
	return p, nil
}

func (r *PostgresOrderRepository) UpdatePayment(ctx context.Context, p *domain.Payment) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE payments
		SET status = $1, payment_method = $2, xendit_refund_id = $3, updated_at = NOW()
		WHERE id = $4
	`, string(p.Status), nullableStr(p.PaymentMethod), nullableStr(p.XenditRefundID), p.ID)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	return nil
}

func (r *PostgresOrderRepository) GetBuyerEmail(ctx context.Context, buyerID string) (string, error) {
	var email string
	err := r.pool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, buyerID).Scan(&email)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("user tidak ditemukan")
	}
	return email, err
}

func (r *PostgresOrderRepository) GetEventDate(ctx context.Context, eventID string) (time.Time, error) {
	var eventDate time.Time
	err := r.pool.QueryRow(ctx, `SELECT date FROM events WHERE id = $1`, eventID).Scan(&eventDate)
	if err != nil {
		return time.Time{}, err
	}
	return eventDate, nil
}

func (r *PostgresOrderRepository) GetRefundRequestsByOrganizer(ctx context.Context, organizerID string) ([]*application.RefundRequestItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 
			o.id,
			o.buyer_id,
			u.email as buyer_email,
			o.event_id,
			e.name as event_name,
			o.status,
			o.total_amount,
			o.created_at
		FROM orders o
		JOIN events e ON e.id = o.event_id
		JOIN users u ON u.id = o.buyer_id
		WHERE e.organizer_id = $1
		  AND o.status IN ('REFUND_REQUESTED', 'REFUND_ORGANIZER_APPROVED')
		ORDER BY o.updated_at DESC
	`, organizerID)
	if err != nil {
		return nil, fmt.Errorf("get refund requests by organizer: %w", err)
	}
	defer rows.Close()

	var list []*application.RefundRequestItem
	for rows.Next() {
		item := &application.RefundRequestItem{}
		if err := rows.Scan(
			&item.OrderID,
			&item.BuyerID,
			&item.BuyerEmail,
			&item.EventID,
			&item.EventName,
			&item.Status,
			&item.TotalAmount,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}


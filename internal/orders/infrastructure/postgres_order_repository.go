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
// Total amount dihitung dari harga ticket_type masing-masing unit dan dipotong promo jika valid.
func (r *PostgresOrderRepository) CreateOrder(ctx context.Context, buyerID, eventID string, unitIDs []string, promoCode string) (*domain.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Pastikan semua unit HELD dan ambil total harga.
	var subtotal int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(tt.price), 0)
		FROM ticket_units tu
		JOIN ticket_types tt ON tt.id = tu.ticket_type_id
		WHERE tu.id = ANY($1) AND tu.status = 'HELD' AND tu.order_id IS NULL
	`, unitIDs).Scan(&subtotal)
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

	// 1. Cek apakah event ini memiliki Promo Event otomatis aktif (PROMO)
	var eventDiscount int64
	var epID string
	var epDiscType string
	var epDiscVal, epMinOrder, epMaxDisc int64
	var epMaxUsage, epUsedCount int
	var epStartDate, epEndDate *time.Time

	err = tx.QueryRow(ctx, `
		SELECT id, discount_type, discount_value, min_order_amount, max_discount_amount, max_usage, used_count, start_date, end_date
		FROM promos
		WHERE type = 'PROMO' AND event_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC LIMIT 1
		FOR UPDATE
	`, eventID).Scan(&epID, &epDiscType, &epDiscVal, &epMinOrder, &epMaxDisc, &epMaxUsage, &epUsedCount, &epStartDate, &epEndDate)

	if err == nil {
		now := time.Now()
		timeValid := (epStartDate == nil || !now.Before(*epStartDate)) && (epEndDate == nil || !now.After(*epEndDate))
		if timeValid && (epMaxUsage == 0 || epUsedCount < epMaxUsage) && subtotal >= epMinOrder {
			if epDiscType == "PERCENTAGE" {
				eventDiscount = (subtotal * epDiscVal) / 100
				if epMaxDisc > 0 && eventDiscount > epMaxDisc {
					eventDiscount = epMaxDisc
				}
			} else {
				eventDiscount = epDiscVal
			}
			if eventDiscount > subtotal {
				eventDiscount = subtotal
			}
			_, _ = tx.Exec(ctx, `UPDATE promos SET used_count = used_count + 1, updated_at = NOW() WHERE id = $1`, epID)
		}
	}

	subtotalAfterPromo := subtotal - eventDiscount

	// 2. Cek apakah pembeli memasukkan kode Voucher Belanja tambahan (VOUCHER)
	var voucherDiscount int64
	var validPromoCode *string
	if promoCode != "" {
		var vID string
		var vDiscType string
		var vDiscVal, vMinOrder, vMaxDisc int64
		var vMaxUsage, vUsedCount int
		var vStartDate, vEndDate *time.Time

		err = tx.QueryRow(ctx, `
			SELECT id, discount_type, discount_value, min_order_amount, max_discount_amount, max_usage, used_count, start_date, end_date
			FROM promos
			WHERE type = 'VOUCHER' AND UPPER(code) = UPPER($1) AND is_active = TRUE
			FOR UPDATE
		`, promoCode).Scan(&vID, &vDiscType, &vDiscVal, &vMinOrder, &vMaxDisc, &vMaxUsage, &vUsedCount, &vStartDate, &vEndDate)

		if err == nil {
			now := time.Now()
			timeValid := (vStartDate == nil || !now.Before(*vStartDate)) && (vEndDate == nil || !now.After(*vEndDate))
			if timeValid && (vMaxUsage == 0 || vUsedCount < vMaxUsage) && subtotalAfterPromo >= vMinOrder {
				if vDiscType == "PERCENTAGE" {
					voucherDiscount = (subtotalAfterPromo * vDiscVal) / 100
					if vMaxDisc > 0 && voucherDiscount > vMaxDisc {
						voucherDiscount = vMaxDisc
					}
				} else {
					voucherDiscount = vDiscVal
				}
				if voucherDiscount > subtotalAfterPromo {
					voucherDiscount = subtotalAfterPromo
				}
				validPromoCode = &promoCode
				_, _ = tx.Exec(ctx, `UPDATE promos SET used_count = used_count + 1, updated_at = NOW() WHERE id = $1`, vID)
			}
		}
	}

	totalDiscount := eventDiscount + voucherDiscount
	finalTotal := subtotal - totalDiscount
	if finalTotal < 0 {
		finalTotal = 0
	}

	order := &domain.Order{}
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (buyer_id, event_id, status, total_amount, promo_code, discount_amount)
		VALUES ($1, $2, 'PENDING', $3, $4, $5)
		RETURNING id, buyer_id, event_id, status, total_amount, COALESCE(promo_code, ''), COALESCE(discount_amount, 0), created_at, updated_at
	`, buyerID, eventID, finalTotal, validPromoCode, totalDiscount).Scan(
		&order.ID, &order.BuyerID, &order.EventID, &order.Status,
		&order.TotalAmount, &order.PromoCode, &order.DiscountAmount, &order.CreatedAt, &order.UpdatedAt,
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
	if len(orderID) != 36 {
		return nil, domain.ErrOrderNotFound
	}
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
			COALESCE(SUM(CASE WHEN tu.status = 'ADMITTED' THEN 1 ELSE 0 END), 0) as admitted_count,
			o.created_at, 
			o.updated_at
		FROM orders o
		LEFT JOIN events e ON e.id = o.event_id
		LEFT JOIN ticket_units tu ON tu.order_id = o.id
		WHERE o.id = $1
		GROUP BY o.id, e.name
	`, orderID).Scan(
		&o.ID, &o.BuyerID, &o.EventID, &o.EventName, &o.Status,
		&o.TotalAmount, &o.UnitIDs, &o.AdmittedCount, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	if o.Status == domain.OrderStatusPending && time.Since(o.CreatedAt) > 5*time.Minute {
		_, _ = r.pool.Exec(ctx, `
			UPDATE orders SET status = 'CANCELLED', updated_at = NOW() 
			WHERE id = $1 AND status = 'PENDING'
		`, o.ID)
		o.Status = domain.OrderStatusCancelled
	}

	return o, nil
}

// ReleaseExpiredHeldOrders membatalkan order yang tidak lagi memiliki tiket berstatus HELD atau sudah lewat 5 menit.
// Digunakan sebelum query get orders agar status PENDING yang expired ter-update ke CANCELLED.
func (r *PostgresOrderRepository) ReleaseExpiredHeldOrders(ctx context.Context, buyerID string) error {
	// 1. Release ticket units yang status HELD dan held_until < NOW()
	_, _ = r.pool.Exec(ctx, `
		UPDATE ticket_units
		SET status = 'AVAILABLE', held_until = NULL, order_id = NULL, updated_at = NOW()
		WHERE status = 'HELD' AND held_until < NOW()
	`)

	// 2. Cancel orders berstatus PENDING yang sudah lewat 5 menit atau tidak punya tiket HELD aktif
	_, err := r.pool.Exec(ctx, `
		UPDATE orders
		SET status = 'CANCELLED', updated_at = NOW()
		WHERE buyer_id = $1
		  AND status = 'PENDING'
		  AND (
		      created_at < NOW() - INTERVAL '5 minutes'
		      OR NOT EXISTS (
		          SELECT 1 FROM ticket_units WHERE order_id = orders.id AND status = 'HELD'
		      )
		  )
	`, buyerID)
	return err
}

func (r *PostgresOrderRepository) GetOrdersByBuyer(ctx context.Context, buyerID string, limit, offset int) ([]*domain.Order, int, error) {
	// Auto-cancel orders yang sudah kadaluarsa sebelum fetch
	_ = r.ReleaseExpiredHeldOrders(ctx, buyerID)

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE buyer_id = $1`, buyerID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders by buyer: %w", err)
	}


	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, `
		SELECT 
			o.id, 
			o.buyer_id, 
			o.event_id, 
			COALESCE(e.name, '') as event_name,
			o.status, 
			o.total_amount, 
			COALESCE(array_agg(tu.id::text) FILTER (WHERE tu.id IS NOT NULL), '{}') as unit_ids,
			COALESCE(SUM(CASE WHEN tu.status = 'ADMITTED' THEN 1 ELSE 0 END), 0) as admitted_count,
			o.created_at, 
			o.updated_at
		FROM orders o
		LEFT JOIN events e ON e.id = o.event_id
		LEFT JOIN ticket_units tu ON tu.order_id = o.id
		WHERE o.buyer_id = $1
		GROUP BY o.id, e.name
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`, buyerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get orders by buyer: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(
			&o.ID, &o.BuyerID, &o.EventID, &o.EventName, &o.Status,
			&o.TotalAmount, &o.UnitIDs, &o.AdmittedCount, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

// ConfirmOrderPayment atomik: update ticket_units HELD→CONFIRMED dan order→PAID.
// Jika 0 unit ter-update (tiket sudah direbut), order diset PAYMENT_DISCREPANCY dan return ErrLostSeat.
func (r *PostgresOrderRepository) ConfirmOrderPayment(ctx context.Context, orderID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Kunci baris order dengan SELECT FOR UPDATE untuk mencegah race condition webhook konkuren
	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM orders WHERE id = $1 FOR UPDATE
	`, orderID).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("lock order: %w", err)
	}

	// 2. Idempotency check: jika status sudah PAID, no-op dan return nil
	if currentStatus == string(domain.OrderStatusPaid) {
		return nil
	}
	if currentStatus == string(domain.OrderStatusPaymentDiscrepancy) {
		return domain.ErrLostSeat
	}

	// 3. Update status ticket_units dari HELD ke CONFIRMED
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
		// Kembalikan tiket ke AVAILABLE agar stok event naik kembali.
		// Status order tetap REFUNDED sebagai audit trail.
		// Tiket yang sudah ADMITTED tidak bisa dikembalikan ke stok (sudah masuk venue).
		_, err = tx.Exec(ctx, `
			UPDATE ticket_units
			SET status = 'AVAILABLE', order_id = NULL, updated_at = NOW()
			WHERE order_id = $1 AND status IN ('HELD', 'CONFIRMED', 'PAYMENT_PENDING')
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

func (r *PostgresOrderRepository) GetRefundRequestsByOrganizer(ctx context.Context, organizerID string, limit, offset int) ([]*application.RefundRequestItem, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM orders o
		JOIN events e ON e.id = o.event_id
		WHERE e.organizer_id = $1
		  AND o.status IN ('REFUND_REQUESTED', 'REFUND_ORGANIZER_APPROVED', 'REFUNDED')
	`, organizerID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count refund requests by organizer: %w", err)
	}

	if limit <= 0 {
		limit = 10
	}

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
		  AND o.status IN ('REFUND_REQUESTED', 'REFUND_ORGANIZER_APPROVED', 'REFUNDED')
		ORDER BY o.updated_at DESC
		LIMIT $2 OFFSET $3
	`, organizerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get refund requests by organizer: %w", err)
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
			return nil, 0, err
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}


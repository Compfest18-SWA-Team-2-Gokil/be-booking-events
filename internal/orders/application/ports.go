package application

import (
	"context"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type OrderRepository interface {
	// CreateOrder membuat order dan mengaitkan unit_ids ke order dalam satu transaksi.
	// Menghitung total_amount dari harga ticket_type masing-masing unit.
	CreateOrder(ctx context.Context, buyerID, eventID string, unitIDs []string) (*domain.Order, error)

	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	// GetOrdersByBuyer mengembalikan seluruh order milik buyer tertentu, paling baru duluan.
	GetOrdersByBuyer(ctx context.Context, buyerID string) ([]*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error

	// ConfirmOrderPayment konfirmasi pembayaran atomik; return ErrLostSeat jika tiket sudah direbut.
	ConfirmOrderPayment(ctx context.Context, orderID string) error
	// HasAdmittedUnits true jika minimal 1 tiket sudah di-scan gerbang (blokir refund).
	HasAdmittedUnits(ctx context.Context, orderID string) (bool, error)

	CreatePayment(ctx context.Context, p *domain.Payment) error
	GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	UpdatePayment(ctx context.Context, p *domain.Payment) error

	// GetBuyerEmail dipakai saat buat Xendit invoice.
	GetBuyerEmail(ctx context.Context, buyerID string) (string, error)

	// GetEventDate mengambil tanggal event untuk validasi batas waktu refund H-1.
	GetEventDate(ctx context.Context, eventID string) (time.Time, error)

	// GetRefundRequestsByOrganizer mengambil daftar permintaan refund untuk event milik organizer.
	GetRefundRequestsByOrganizer(ctx context.Context, organizerID string) ([]*RefundRequestItem, error)
}

type RefundRequestItem struct {
	OrderID     string             `json:"order_id"`
	BuyerID     string             `json:"buyer_id"`
	BuyerEmail  string             `json:"buyer_email"`
	EventID     string             `json:"event_id"`
	EventName   string             `json:"event_name"`
	Status      domain.OrderStatus `json:"status"`
	TotalAmount int64              `json:"total_amount"`
	CreatedAt   time.Time          `json:"created_at"`
}

// PaymentProvider abstraksi ke payment gateway (Xendit).
type PaymentProvider interface {
	CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*InvoiceResult, error)
	RefundPayment(ctx context.Context, invoiceID string, amount int64) (refundID string, err error)
}

type CreateInvoiceInput struct {
	ExternalID  string // order ID kita sebagai reference
	Amount      int64
	PayerEmail  string
	Description string
}

type InvoiceResult struct {
	InvoiceID  string
	InvoiceURL string
}

package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type OrderRepository interface {
	// CreateOrder membuat order dan mengaitkan unit_ids ke order dalam satu transaksi.
	// Menghitung total_amount dari harga ticket_type masing-masing unit.
	CreateOrder(ctx context.Context, buyerID, eventID string, unitIDs []string) (*domain.Order, error)

	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error

	CreatePayment(ctx context.Context, p *domain.Payment) error
	GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	UpdatePayment(ctx context.Context, p *domain.Payment) error

	// GetBuyerEmail dipakai saat buat Xendit invoice.
	GetBuyerEmail(ctx context.Context, buyerID string) (string, error)
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

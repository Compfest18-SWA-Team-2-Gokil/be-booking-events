package domain

import "time"

type OrderStatus string
type PaymentStatus string

const (
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusPaymentPending  OrderStatus = "PAYMENT_PENDING"
	OrderStatusPaid            OrderStatus = "PAID"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
	OrderStatusRefundRequested OrderStatus = "REFUND_REQUESTED"
	OrderStatusRefunded        OrderStatus = "REFUNDED"

	PaymentStatusPending  PaymentStatus = "PENDING"
	PaymentStatusSuccess  PaymentStatus = "SUCCESS"
	PaymentStatusFailed   PaymentStatus = "FAILED"
	PaymentStatusRefunded PaymentStatus = "REFUNDED"
)

type Order struct {
	ID          string
	BuyerID     string
	EventID     string
	Status      OrderStatus
	TotalAmount int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Payment struct {
	ID               string
	OrderID          string
	Amount           int64
	Status           PaymentStatus
	PaymentMethod    string
	XenditInvoiceID  string
	XenditInvoiceURL string
	XenditRefundID   string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

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
	ID          string      `json:"id"`
	BuyerID     string      `json:"buyer_id"`
	EventID     string      `json:"event_id"`
	Status      OrderStatus `json:"status"`
	TotalAmount int64       `json:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Payment struct {
	ID               string        `json:"id"`
	OrderID          string        `json:"order_id"`
	Amount           int64         `json:"amount"`
	Status           PaymentStatus `json:"status"`
	PaymentMethod    string        `json:"payment_method,omitempty"`
	XenditInvoiceID  string        `json:"xendit_invoice_id,omitempty"`
	XenditInvoiceURL string        `json:"xendit_invoice_url,omitempty"`
	XenditRefundID   string        `json:"xendit_refund_id,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

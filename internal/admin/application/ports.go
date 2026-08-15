package application

import (
	"context"
	"time"
)

// DisputeOrder representasi order dengan anomali untuk dispute dashboard.
type DisputeOrder struct {
	OrderID     string    `json:"order_id"`
	BuyerID     string    `json:"buyer_id"`
	BuyerEmail  string    `json:"buyer_email"`
	EventID     string    `json:"event_id"`
	Status      string    `json:"status"`
	TotalAmount int64     `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AdminRepository interface {
	// ListDisputes mengembalikan order dengan status anomali (PAYMENT_DISCREPANCY, dll).
	ListDisputes(ctx context.Context) ([]DisputeOrder, error)
	// OverrideOrderStatus mengubah status order secara paksa dan mencatat ke audit_log.
	OverrideOrderStatus(ctx context.Context, orderID, adminID, newStatus, reason string) error
}

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

// AuditLogEntry representasi immutable record perubahan status / tindakan admin.
type AuditLogEntry struct {
	ID         string    `json:"id"`
	ActorID    string    `json:"actor_id"`
	ActorEmail string    `json:"actor_email"`
	ActorRole  string    `json:"actor_role"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Action     string    `json:"action"`
	FromStatus *string   `json:"from_status,omitempty"`
	ToStatus   *string   `json:"to_status,omitempty"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

type AdminRepository interface {
	// ListDisputes mengembalikan order dengan status anomali (PAYMENT_DISCREPANCY, dll).
	ListDisputes(ctx context.Context) ([]DisputeOrder, error)
	// OverrideOrderStatus mengubah status order secara paksa dan mencatat ke audit_log.
	OverrideOrderStatus(ctx context.Context, orderID, adminID, newStatus, reason string) error
	// ListAuditLogs mengambil riwayat immutable audit trail logs.
	ListAuditLogs(ctx context.Context, limit int) ([]AuditLogEntry, error)
}

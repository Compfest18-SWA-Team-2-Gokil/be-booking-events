package application

import (
	"context"

	auditdomain "github.com/ebk-tech/be-booking-events/internal/audit/domain"
	ordersdomain "github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type AdminRepository interface {
	ListDisputedOrders(ctx context.Context) ([]*ordersdomain.Order, error)
	OverrideOrderStatus(ctx context.Context, orderID string, newStatus ordersdomain.OrderStatus, reason string, actorID string) error
	ReassignTicket(ctx context.Context, unitID string, targetOrderID string, newSeatNumber string, reason string, actorID string) error
	ListAuditLogs(ctx context.Context, limit int) ([]*auditdomain.AuditLog, error)
}

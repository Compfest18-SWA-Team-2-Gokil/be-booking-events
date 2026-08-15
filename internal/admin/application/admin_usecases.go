package application

import (
	"context"

	auditdomain "github.com/ebk-tech/be-booking-events/internal/audit/domain"
	ordersdomain "github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type AdminUseCases struct {
	repo AdminRepository
}

func NewAdminUseCases(repo AdminRepository) *AdminUseCases {
	return &AdminUseCases{repo: repo}
}

func (uc *AdminUseCases) ListDisputes(ctx context.Context) ([]*ordersdomain.Order, error) {
	return uc.repo.ListDisputedOrders(ctx)
}

type OverrideOrderInput struct {
	OrderID   string
	NewStatus ordersdomain.OrderStatus
	Reason    string
	ActorID   string
}

func (uc *AdminUseCases) OverrideOrderStatus(ctx context.Context, input OverrideOrderInput) error {
	return uc.repo.OverrideOrderStatus(ctx, input.OrderID, input.NewStatus, input.Reason, input.ActorID)
}

type ReassignTicketInput struct {
	UnitID        string
	TargetOrderID string
	NewSeatNumber string
	Reason        string
	ActorID       string
}

func (uc *AdminUseCases) ReassignTicket(ctx context.Context, input ReassignTicketInput) error {
	return uc.repo.ReassignTicket(ctx, input.UnitID, input.TargetOrderID, input.NewSeatNumber, input.Reason, input.ActorID)
}

func (uc *AdminUseCases) ListAuditLogs(ctx context.Context, limit int) ([]*auditdomain.AuditLog, error) {
	return uc.repo.ListAuditLogs(ctx, limit)
}

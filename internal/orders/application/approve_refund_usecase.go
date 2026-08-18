package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/audit"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type ApproveRefundUseCase struct {
	repo        OrderRepository
	provider    PaymentProvider
	auditLogger *audit.Logger
}

func NewApproveRefundUseCase(repo OrderRepository, provider PaymentProvider, auditLogger *audit.Logger) *ApproveRefundUseCase {
	return &ApproveRefundUseCase{repo: repo, provider: provider, auditLogger: auditLogger}
}

func (uc *ApproveRefundUseCase) Execute(ctx context.Context, orderID string) error {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != domain.OrderStatusRefundRequested {
		return domain.ErrRefundNotRequested
	}

	// Organizer menyetujui refund -> status berubah jadi REFUND_ORGANIZER_APPROVED (diteruskan ke Admin untuk final approval)
	if err := uc.repo.UpdateOrderStatus(ctx, orderID, domain.OrderStatusRefundOrganizerApproved); err != nil {
		return err
	}

	// Audit: organizer approved refund
	if uc.auditLogger != nil {
		uc.auditLogger.Log(ctx, audit.Entry{
			ActorID:    "",
			ActorRole:  "ORGANIZER",
			EntityType: "order",
			EntityID:   orderID,
			Action:     "REFUND_APPROVED",
			FromStatus: string(domain.OrderStatusRefundRequested),
			ToStatus:   string(domain.OrderStatusRefundOrganizerApproved),
		})
	}

	return nil
}

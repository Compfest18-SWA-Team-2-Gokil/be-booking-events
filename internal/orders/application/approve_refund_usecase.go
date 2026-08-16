package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type ApproveRefundUseCase struct {
	repo     OrderRepository
	provider PaymentProvider
}

func NewApproveRefundUseCase(repo OrderRepository, provider PaymentProvider) *ApproveRefundUseCase {
	return &ApproveRefundUseCase{repo: repo, provider: provider}
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
	return uc.repo.UpdateOrderStatus(ctx, orderID, domain.OrderStatusRefundOrganizerApproved)
}

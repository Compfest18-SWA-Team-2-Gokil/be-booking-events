package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type RequestRefundUseCase struct {
	repo OrderRepository
}

func NewRequestRefundUseCase(repo OrderRepository) *RequestRefundUseCase {
	return &RequestRefundUseCase{repo: repo}
}

func (uc *RequestRefundUseCase) Execute(ctx context.Context, orderID, buyerID string) error {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	if order.BuyerID != buyerID {
		return domain.ErrOrderNotFound
	}

	if order.Status != domain.OrderStatusPaid {
		return domain.ErrOrderNotPaid
	}

	// PRD-09 Status Blocker: tolak refund jika ada tiket yang sudah di-scan gerbang.
	admitted, err := uc.repo.HasAdmittedUnits(ctx, orderID)
	if err != nil {
		return err
	}
	if admitted {
		return domain.ErrTicketAlreadyAdmitted
	}

	return uc.repo.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusRefundRequested)
}

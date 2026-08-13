package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type GetOrderUseCase struct {
	repo OrderRepository
}

func NewGetOrderUseCase(repo OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{repo: repo}
}

func (uc *GetOrderUseCase) Execute(ctx context.Context, orderID, buyerID string) (*domain.Order, error) {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerID != buyerID {
		return nil, domain.ErrOrderNotFound
	}
	return order, nil
}

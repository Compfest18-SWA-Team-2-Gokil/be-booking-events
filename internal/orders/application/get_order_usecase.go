package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
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

// ExecuteByBuyer mengambil seluruh riwayat order milik buyer dengan pagination.
func (uc *GetOrderUseCase) ExecuteByBuyer(ctx context.Context, buyerID string, limit, offset int) ([]*domain.Order, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return uc.repo.GetOrdersByBuyer(ctx, buyerID, limit, offset)
}

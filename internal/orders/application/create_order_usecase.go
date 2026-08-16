package application

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type CreateOrderUseCase struct {
	repo OrderRepository
}

func NewCreateOrderUseCase(repo OrderRepository) *CreateOrderUseCase {
	return &CreateOrderUseCase{repo: repo}
}

type CreateOrderInput struct {
	BuyerID string
	EventID string
	UnitIDs []string
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	if len(input.UnitIDs) == 0 {
		return nil, fmt.Errorf("%w", domain.ErrNoHeldUnits)
	}

	order, err := uc.repo.CreateOrder(ctx, input.BuyerID, input.EventID, input.UnitIDs)
	if err != nil {
		return nil, err
	}

	return order, nil
}

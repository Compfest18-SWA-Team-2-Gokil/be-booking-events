package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type GetEventUseCase struct {
	repo EventRepository
}

func NewGetEventUseCase(repo EventRepository) *GetEventUseCase {
	return &GetEventUseCase{repo: repo}
}

func (uc *GetEventUseCase) Execute(ctx context.Context, eventID string) (*domain.Event, error) {
	return uc.repo.GetEvent(ctx, eventID)
}

package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type ListEventsUseCase struct {
	repo EventRepository
}

func NewListEventsUseCase(repo EventRepository) *ListEventsUseCase {
	return &ListEventsUseCase{repo: repo}
}

func (uc *ListEventsUseCase) Execute(ctx context.Context) ([]*domain.Event, error) {
	return uc.repo.ListEvents(ctx)
}

type ListTicketTypesUseCase struct {
	repo EventRepository
}

func NewListTicketTypesUseCase(repo EventRepository) *ListTicketTypesUseCase {
	return &ListTicketTypesUseCase{repo: repo}
}

func (uc *ListTicketTypesUseCase) Execute(ctx context.Context, eventID string) ([]*domain.TicketType, error) {
	if eventID == "" {
		return nil, domain.ErrEventNotFound
	}
	return uc.repo.ListTicketTypes(ctx, eventID)
}

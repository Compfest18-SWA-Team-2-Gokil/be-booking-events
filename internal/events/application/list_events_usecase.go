package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/events/domain"
)

type ListEventsUseCase struct {
	repo EventRepository
}

func NewListEventsUseCase(repo EventRepository) *ListEventsUseCase {
	return &ListEventsUseCase{repo: repo}
}

func (uc *ListEventsUseCase) Execute(ctx context.Context, filter ListEventsFilter) ([]*domain.Event, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	return uc.repo.ListEvents(ctx, filter)
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

package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type UpdateEventUseCase struct {
	repo EventRepository
}

func NewUpdateEventUseCase(repo EventRepository) *UpdateEventUseCase {
	return &UpdateEventUseCase{repo: repo}
}

type UpdateEventInput struct {
	EventID     string
	OrganizerID string
	Name        string
	Description string
	Category    string
	Date        string // RFC3339
	Location    string
}

func (uc *UpdateEventUseCase) Execute(ctx context.Context, input UpdateEventInput) (*domain.Event, error) {
	event, err := uc.repo.GetEvent(ctx, input.EventID)
	if err != nil {
		return nil, err
	}
	if event.OrganizerID != input.OrganizerID {
		return nil, domain.ErrNotEventOrganizer
	}

	date, err := parseDate(input.Date)
	if err != nil {
		return nil, err
	}

	event.Name = input.Name
	event.Description = input.Description
	event.Category = domain.Category(input.Category)
	event.Date = date
	event.Location = input.Location

	if err := event.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

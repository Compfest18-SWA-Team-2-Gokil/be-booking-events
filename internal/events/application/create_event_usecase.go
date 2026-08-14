package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type CreateEventUseCase struct {
	repo EventRepository
}

func NewCreateEventUseCase(repo EventRepository) *CreateEventUseCase {
	return &CreateEventUseCase{repo: repo}
}

type CreateEventInput struct {
	OrganizerID string
	Name        string
	Description string
	Category    string
	Date        string // RFC3339
	Location    string
}

func (uc *CreateEventUseCase) Execute(ctx context.Context, input CreateEventInput) (*domain.Event, error) {
	date, err := parseDate(input.Date)
	if err != nil {
		return nil, err
	}

	event := &domain.Event{
		OrganizerID: input.OrganizerID,
		Name:        input.Name,
		Description: input.Description,
		Category:    domain.Category(input.Category),
		Date:        date,
		Location:    input.Location,
	}

	if err := event.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateEvent(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

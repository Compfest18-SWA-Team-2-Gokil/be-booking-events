package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type DeleteEventUseCase struct {
	repo EventRepository
}

func NewDeleteEventUseCase(repo EventRepository) *DeleteEventUseCase {
	return &DeleteEventUseCase{repo: repo}
}

func (uc *DeleteEventUseCase) Execute(ctx context.Context, eventID, organizerID string) error {
	event, err := uc.repo.GetEvent(ctx, eventID)
	if err != nil {
		return err
	}
	if event.OrganizerID != organizerID {
		return domain.ErrNotEventOrganizer
	}

	hasSold, err := uc.repo.HasNonAvailableUnits(ctx, eventID)
	if err != nil {
		return err
	}
	if hasSold {
		return domain.ErrCannotDeleteWithSoldTickets
	}

	return uc.repo.DeleteEvent(ctx, eventID)
}

package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type DeleteTicketTypeUseCase struct {
	repo EventRepository
}

func NewDeleteTicketTypeUseCase(repo EventRepository) *DeleteTicketTypeUseCase {
	return &DeleteTicketTypeUseCase{repo: repo}
}

func (uc *DeleteTicketTypeUseCase) Execute(ctx context.Context, ticketTypeID, organizerID string) error {
	tt, err := uc.repo.GetTicketType(ctx, ticketTypeID)
	if err != nil {
		return err
	}

	event, err := uc.repo.GetEvent(ctx, tt.EventID)
	if err != nil {
		return err
	}
	if event.OrganizerID != organizerID {
		return domain.ErrNotEventOrganizer
	}

	sold, err := uc.repo.CountSoldUnits(ctx, ticketTypeID)
	if err != nil {
		return err
	}
	if sold > 0 {
		return domain.ErrCannotDeleteTTWithSoldTickets
	}

	return uc.repo.DeleteTicketType(ctx, ticketTypeID)
}

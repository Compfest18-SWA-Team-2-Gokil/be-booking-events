package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/events/domain"
)

type CreateTicketTypeUseCase struct {
	repo EventRepository
}

func NewCreateTicketTypeUseCase(repo EventRepository) *CreateTicketTypeUseCase {
	return &CreateTicketTypeUseCase{repo: repo}
}

type CreateTicketTypeInput struct {
	EventID    string
	Name       string
	Price      int64
	Kind       string
	TotalQuota int
}

func (uc *CreateTicketTypeUseCase) Execute(ctx context.Context, input CreateTicketTypeInput) (*domain.TicketType, error) {
	// Pastikan event ada.
	if _, err := uc.repo.GetEvent(ctx, input.EventID); err != nil {
		return nil, domain.ErrEventNotFound
	}

	tt := &domain.TicketType{
		EventID:    input.EventID,
		Name:       input.Name,
		Price:      input.Price,
		Kind:       domain.Kind(input.Kind),
		TotalQuota: input.TotalQuota,
	}

	if err := tt.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateTicketType(ctx, tt); err != nil {
		return nil, err
	}
	return tt, nil
}

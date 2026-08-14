package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type UpdateTicketTypeUseCase struct {
	repo EventRepository
}

func NewUpdateTicketTypeUseCase(repo EventRepository) *UpdateTicketTypeUseCase {
	return &UpdateTicketTypeUseCase{repo: repo}
}

type UpdateTicketTypeInput struct {
	TicketTypeID string
	OrganizerID  string
	Name         string
	Price        int64
	TotalQuota   int
}

func (uc *UpdateTicketTypeUseCase) Execute(ctx context.Context, input UpdateTicketTypeInput) (*domain.TicketType, error) {
	tt, err := uc.repo.GetTicketType(ctx, input.TicketTypeID)
	if err != nil {
		return nil, err
	}

	// Cek kepemilikan event.
	event, err := uc.repo.GetEvent(ctx, tt.EventID)
	if err != nil {
		return nil, err
	}
	if event.OrganizerID != input.OrganizerID {
		return nil, domain.ErrNotEventOrganizer
	}

	// PRD-02: harga dikunci jika ada pesanan berbayar.
	if tt.PriceStatus == "LOCKED" && input.Price != tt.Price {
		return nil, domain.ErrPriceLocked
	}

	// PRD-02: kuota hanya bisa diturunkan sampai jumlah terjual.
	if input.TotalQuota < tt.TotalQuota {
		sold, err := uc.repo.CountSoldUnits(ctx, input.TicketTypeID)
		if err != nil {
			return nil, err
		}
		if input.TotalQuota < sold {
			return nil, domain.ErrQuotaBelowSold
		}
	}

	tt.Name = input.Name
	tt.Price = input.Price
	tt.TotalQuota = input.TotalQuota

	if err := uc.repo.UpdateTicketType(ctx, tt); err != nil {
		return nil, err
	}
	return tt, nil
}

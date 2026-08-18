package application

import (
	"context"
	"fmt"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/inventory/domain"
)

// HoldDuration: 5 menit fixed, tidak configurable per-event di MVP (keputusan TDD Bab 4).
const HoldDuration = 5 * time.Minute

type HoldTicketUseCase struct {
	repo TicketUnitRepository
}

func NewHoldTicketUseCase(repo TicketUnitRepository) *HoldTicketUseCase {
	return &HoldTicketUseCase{repo: repo}
}

type HoldTicketInput struct {
	Items []HoldRequest // satu order bisa span beberapa TicketType (Open Question #2 resolved)
}

type HoldTicketOutput struct {
	Units     []*domain.TicketUnit
	HeldUntil time.Time
}

func (uc *HoldTicketUseCase) Execute(ctx context.Context, input HoldTicketInput) (*HoldTicketOutput, error) {
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("hold request harus memiliki minimal satu item")
	}
	for _, item := range input.Items {
		if item.Quantity < 1 {
			return nil, domain.ErrTicketNotAvailable
		}
	}

	holdUntil := time.Now().Add(HoldDuration)
	units, err := uc.repo.HoldAvailableUnits(ctx, input.Items, holdUntil)
	if err != nil {
		return nil, err
	}

	return &HoldTicketOutput{Units: units, HeldUntil: holdUntil}, nil
}

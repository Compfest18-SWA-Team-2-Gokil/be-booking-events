package application

import (
	"context"
	"fmt"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type ProvisionUnitsUseCase struct {
	repo EventRepository
}

func NewProvisionUnitsUseCase(repo EventRepository) *ProvisionUnitsUseCase {
	return &ProvisionUnitsUseCase{repo: repo}
}

type ProvisionUnitsInput struct {
	TicketTypeID string
	Quantity     int
}

type ProvisionUnitsOutput struct {
	TicketTypeID string `json:"ticket_type_id"`
	Provisioned  int    `json:"provisioned"`
	Total        int    `json:"total"`
	ProvisionedAt time.Time `json:"provisioned_at"`
}

func (uc *ProvisionUnitsUseCase) Execute(ctx context.Context, input ProvisionUnitsInput) (*ProvisionUnitsOutput, error) {
	if input.Quantity < 1 {
		return nil, fmt.Errorf("quantity harus minimal 1")
	}

	tt, err := uc.repo.GetTicketType(ctx, input.TicketTypeID)
	if err != nil {
		return nil, domain.ErrTicketTypeNotFound
	}

	provisioned, err := uc.repo.CountProvisionedUnits(ctx, input.TicketTypeID)
	if err != nil {
		return nil, err
	}

	remaining := tt.TotalQuota - provisioned
	if input.Quantity > remaining {
		return nil, fmt.Errorf("%w: sisa kuota %d, diminta %d", domain.ErrQuotaExceeded, remaining, input.Quantity)
	}

	if err := uc.repo.ProvisionUnits(ctx, input.TicketTypeID, input.Quantity); err != nil {
		return nil, err
	}

	return &ProvisionUnitsOutput{
		TicketTypeID: input.TicketTypeID,
		Provisioned:  input.Quantity,
		Total:        provisioned + input.Quantity,
		ProvisionedAt: time.Now(),
	}, nil
}

package application

import (
	"context"
	"log/slog"
)

type ExpireHeldTicketsUseCase struct {
	repo TicketUnitRepository
}

func NewExpireHeldTicketsUseCase(repo TicketUnitRepository) *ExpireHeldTicketsUseCase {
	return &ExpireHeldTicketsUseCase{repo: repo}
}

func (uc *ExpireHeldTicketsUseCase) Execute(ctx context.Context) error {
	count, err := uc.repo.ExpireHeldUnits(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Info("expired held ticket units", "count", count)
	}
	return nil
}

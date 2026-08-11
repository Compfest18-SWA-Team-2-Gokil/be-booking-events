package application

import (
	"context"
	"fmt"
)

type GetEventMetricsUseCase struct {
	repo MetricsRepository
}

func NewGetEventMetricsUseCase(repo MetricsRepository) *GetEventMetricsUseCase {
	return &GetEventMetricsUseCase{repo: repo}
}

func (uc *GetEventMetricsUseCase) Execute(ctx context.Context, eventID string) ([]TicketTypeMetrics, error) {
	if eventID == "" {
		return nil, fmt.Errorf("eventID harus diisi")
	}
	return uc.repo.GetEventMetrics(ctx, eventID)
}

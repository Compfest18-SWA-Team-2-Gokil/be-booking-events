package application_test

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/dashboard/application"
)

type fakeMetricsRepo struct {
	data []application.TicketTypeMetrics
	err  error
}

func (r *fakeMetricsRepo) GetEventMetrics(_ context.Context, _ string) ([]application.TicketTypeMetrics, error) {
	return r.data, r.err
}

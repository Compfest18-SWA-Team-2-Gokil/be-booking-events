package application_test

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/dashboard/application"
)

type fakeMetricsRepo struct {
	data []application.TicketTypeMetrics
	err  error
}

func (r *fakeMetricsRepo) GetEventMetrics(_ context.Context, _ string) ([]application.TicketTypeMetrics, error) {
	return r.data, r.err
}

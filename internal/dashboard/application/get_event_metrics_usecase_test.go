package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ebk-tech/be-booking-events/internal/dashboard/application"
)

func TestGetEventMetricsUseCase_Execute_Success(t *testing.T) {
	fake := &fakeMetricsRepo{
		data: []application.TicketTypeMetrics{
			{TicketTypeID: "type-1", Available: 50, Held: 10, Sold: 30, Admitted: 5, Refunded: 2, Total: 97},
		},
	}
	uc := application.NewGetEventMetricsUseCase(fake)

	result, err := uc.Execute(context.Background(), "event-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(result))
	}
	m := result[0]
	if m.Available != 50 {
		t.Errorf("available = %d, want 50", m.Available)
	}
	if m.Sold != 30 {
		t.Errorf("sold = %d, want 30", m.Sold)
	}
	if m.Total != 97 {
		t.Errorf("total = %d, want 97", m.Total)
	}
}

func TestGetEventMetricsUseCase_Execute_EmptyEventID(t *testing.T) {
	uc := application.NewGetEventMetricsUseCase(&fakeMetricsRepo{})

	_, err := uc.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected error untuk eventID kosong, got nil")
	}
}

func TestGetEventMetricsUseCase_Execute_RepoError(t *testing.T) {
	repoErr := errors.New("db error")
	fake := &fakeMetricsRepo{err: repoErr}
	uc := application.NewGetEventMetricsUseCase(fake)

	_, err := uc.Execute(context.Background(), "event-123")
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestGetEventMetricsUseCase_Execute_EmptyResult(t *testing.T) {
	uc := application.NewGetEventMetricsUseCase(&fakeMetricsRepo{data: nil})

	result, err := uc.Execute(context.Background(), "event-no-tickets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

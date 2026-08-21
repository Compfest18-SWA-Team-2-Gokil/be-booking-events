package application_test

import (
	"context"
	"testing"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

func TestListMyAssignedEventsUseCase(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	uc := application.NewListMyAssignedEventsUseCase(repo)

	// Create gate operator user
	gateUser := &domain.User{
		Email:    "gate@test.com",
		Username: "gateop1",
		Name:     "Gate Operator One",
		Role:     domain.RoleGateOperator,
	}
	if err := repo.Create(ctx, gateUser); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 1. Initial: No assignments
	events, err := uc.Execute(ctx, gateUser.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}

	// 2. Assign to Event 1 and Event 2
	if err := repo.AssignGateOperator(ctx, gateUser.ID, "event-1", "org-1"); err != nil {
		t.Fatalf("failed to assign event-1: %v", err)
	}
	if err := repo.AssignGateOperator(ctx, gateUser.ID, "event-2", "org-1"); err != nil {
		t.Fatalf("failed to assign event-2: %v", err)
	}

	events, err = uc.Execute(ctx, gateUser.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 assigned events, got %d", len(events))
	}

	// 3. Revoke Event 1
	if err := repo.RemoveGateOperator(ctx, gateUser.ID, "event-1"); err != nil {
		t.Fatalf("failed to remove event-1: %v", err)
	}

	events, err = uc.Execute(ctx, gateUser.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 assigned event, got %d", len(events))
	}
	if events[0].EventID != "event-2" {
		t.Errorf("expected event-2, got %s", events[0].EventID)
	}
}

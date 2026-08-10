package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/inventory/application"
	"github.com/ebk-tech/be-booking-events/internal/inventory/domain"
)

func TestExpireHeldTicketsUseCase_Execute_ReleasesExpiredUnits(t *testing.T) {
	past := time.Now().Add(-1 * time.Minute)
	expiredUnit := &domain.TicketUnit{
		ID:        "unit-expired",
		Status:    domain.StatusHeld,
		HeldUntil: &past,
	}
	uc := application.NewExpireHeldTicketsUseCase(newFakeRepo(expiredUnit))

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if expiredUnit.Status != domain.StatusAvailable {
		t.Errorf("expired unit status = %s, want AVAILABLE", expiredUnit.Status)
	}
	if expiredUnit.HeldUntil != nil {
		t.Error("held_until harus nil setelah expire")
	}
}

func TestExpireHeldTicketsUseCase_Execute_KeepsActiveHolds(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	activeUnit := &domain.TicketUnit{
		ID:        "unit-active",
		Status:    domain.StatusHeld,
		HeldUntil: &future,
	}
	uc := application.NewExpireHeldTicketsUseCase(newFakeRepo(activeUnit))

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if activeUnit.Status != domain.StatusHeld {
		t.Errorf("active unit status = %s, want HELD (tidak boleh direlease)", activeUnit.Status)
	}
}

func TestExpireHeldTicketsUseCase_Execute_MixedUnits(t *testing.T) {
	past := time.Now().Add(-1 * time.Minute)
	future := time.Now().Add(5 * time.Minute)

	expiredUnit := &domain.TicketUnit{ID: "unit-expired", Status: domain.StatusHeld, HeldUntil: &past}
	activeUnit := &domain.TicketUnit{ID: "unit-active", Status: domain.StatusHeld, HeldUntil: &future}
	confirmedUnit := &domain.TicketUnit{ID: "unit-confirmed", Status: domain.StatusConfirmed}

	uc := application.NewExpireHeldTicketsUseCase(newFakeRepo(expiredUnit, activeUnit, confirmedUnit))

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatal(err)
	}

	if expiredUnit.Status != domain.StatusAvailable {
		t.Errorf("expired unit harus AVAILABLE, got %s", expiredUnit.Status)
	}
	if activeUnit.Status != domain.StatusHeld {
		t.Errorf("active unit harus tetap HELD, got %s", activeUnit.Status)
	}
	if confirmedUnit.Status != domain.StatusConfirmed {
		t.Errorf("confirmed unit harus tetap CONFIRMED, got %s", confirmedUnit.Status)
	}
}

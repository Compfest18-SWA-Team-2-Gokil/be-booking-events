package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/inventory/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/inventory/domain"
)

func TestHoldTicketUseCase_Execute_SingleType_Success(t *testing.T) {
	typeID := "type-festival"
	unit := &domain.TicketUnit{ID: "unit-1", TicketTypeID: typeID, Status: domain.StatusAvailable}
	uc := application.NewHoldTicketUseCase(newFakeRepo(unit))

	result, err := uc.Execute(context.Background(), application.HoldTicketInput{
		Items: []application.HoldRequest{{TicketTypeID: typeID, Quantity: 1}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(result.Units))
	}
	if result.Units[0].Status != domain.StatusHeld {
		t.Errorf("unit status = %s, want HELD", result.Units[0].Status)
	}
	if result.HeldUntil.Before(time.Now()) {
		t.Error("held_until harus di masa depan")
	}
}

func TestHoldTicketUseCase_Execute_NoUnitsAvailable(t *testing.T) {
	typeID := "type-festival"
	heldUnit := &domain.TicketUnit{ID: "unit-1", TicketTypeID: typeID, Status: domain.StatusHeld}
	uc := application.NewHoldTicketUseCase(newFakeRepo(heldUnit))

	_, err := uc.Execute(context.Background(), application.HoldTicketInput{
		Items: []application.HoldRequest{{TicketTypeID: typeID, Quantity: 1}},
	})

	if !errors.Is(err, domain.ErrTicketNotAvailable) {
		t.Errorf("expected ErrTicketNotAvailable, got %v", err)
	}
}

func TestHoldTicketUseCase_Execute_InsufficientQuantity(t *testing.T) {
	typeID := "type-festival"
	// Hanya 1 unit tersedia, request 2
	unit := &domain.TicketUnit{ID: "unit-1", TicketTypeID: typeID, Status: domain.StatusAvailable}
	uc := application.NewHoldTicketUseCase(newFakeRepo(unit))

	_, err := uc.Execute(context.Background(), application.HoldTicketInput{
		Items: []application.HoldRequest{{TicketTypeID: typeID, Quantity: 2}},
	})

	if !errors.Is(err, domain.ErrTicketNotAvailable) {
		t.Errorf("expected ErrTicketNotAvailable, got %v", err)
	}
}

func TestHoldTicketUseCase_Execute_MultiType_Success(t *testing.T) {
	festUnit := &domain.TicketUnit{ID: "fest-1", TicketTypeID: "type-festival", Status: domain.StatusAvailable}
	vipUnit := &domain.TicketUnit{ID: "vip-1", TicketTypeID: "type-vip", Status: domain.StatusAvailable}
	uc := application.NewHoldTicketUseCase(newFakeRepo(festUnit, vipUnit))

	result, err := uc.Execute(context.Background(), application.HoldTicketInput{
		Items: []application.HoldRequest{
			{TicketTypeID: "type-festival", Quantity: 1},
			{TicketTypeID: "type-vip", Quantity: 1},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Units) != 2 {
		t.Errorf("expected 2 units, got %d", len(result.Units))
	}
}

// Skenario All-or-Nothing: satu tipe gagal → seluruh order dibatalkan.
func TestHoldTicketUseCase_Execute_MultiType_AllOrNothing(t *testing.T) {
	festUnit := &domain.TicketUnit{ID: "fest-1", TicketTypeID: "type-festival", Status: domain.StatusAvailable}
	// VIP tidak ada
	uc := application.NewHoldTicketUseCase(newFakeRepo(festUnit))

	_, err := uc.Execute(context.Background(), application.HoldTicketInput{
		Items: []application.HoldRequest{
			{TicketTypeID: "type-festival", Quantity: 1},
			{TicketTypeID: "type-vip", Quantity: 1},
		},
	})

	if !errors.Is(err, domain.ErrTicketNotAvailable) {
		t.Errorf("expected ErrTicketNotAvailable, got %v", err)
	}
	// Festival unit harus tetap AVAILABLE — rollback karena VIP gagal.
	if festUnit.Status != domain.StatusAvailable {
		t.Errorf("festival unit status = %s setelah failed multi-type hold, want AVAILABLE", festUnit.Status)
	}
}

func TestHoldTicketUseCase_Execute_EmptyItems(t *testing.T) {
	uc := application.NewHoldTicketUseCase(newFakeRepo())

	_, err := uc.Execute(context.Background(), application.HoldTicketInput{Items: nil})

	if err == nil {
		t.Error("expected error untuk empty items, got nil")
	}
}

func TestHoldTicketUseCase_Execute_ZeroQuantity(t *testing.T) {
	uc := application.NewHoldTicketUseCase(newFakeRepo())

	_, err := uc.Execute(context.Background(), application.HoldTicketInput{
		Items: []application.HoldRequest{{TicketTypeID: "type-x", Quantity: 0}},
	})

	if !errors.Is(err, domain.ErrTicketNotAvailable) {
		t.Errorf("expected ErrTicketNotAvailable, got %v", err)
	}
}

func TestHoldTicketUseCase_HoldDuration(t *testing.T) {
	typeID := "type-festival"
	unit := &domain.TicketUnit{ID: "unit-1", TicketTypeID: typeID, Status: domain.StatusAvailable}
	uc := application.NewHoldTicketUseCase(newFakeRepo(unit))

	before := time.Now()
	result, err := uc.Execute(context.Background(), application.HoldTicketInput{
		Items: []application.HoldRequest{{TicketTypeID: typeID, Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedMin := before.Add(application.HoldDuration - time.Second)
	expectedMax := before.Add(application.HoldDuration + time.Second)

	if result.HeldUntil.Before(expectedMin) || result.HeldUntil.After(expectedMax) {
		t.Errorf("held_until = %v, want sekitar %v (±1s)", result.HeldUntil, before.Add(application.HoldDuration))
	}
}

package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/events/application"
	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

var futureDate = time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)

func TestCreateEventUseCase_Execute_Success(t *testing.T) {
	uc := application.NewCreateEventUseCase(newFakeEventRepo())

	event, err := uc.Execute(context.Background(), application.CreateEventInput{
		OrganizerID: "org-1",
		Name:        "Konser Jakarta",
		Date:        futureDate,
		Location:    "GBK Jakarta",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Name != "Konser Jakarta" {
		t.Errorf("name = %s, want Konser Jakarta", event.Name)
	}
	if event.OrganizerID != "org-1" {
		t.Errorf("organizer_id = %s, want org-1", event.OrganizerID)
	}
}

func TestCreateEventUseCase_Execute_DateInPast(t *testing.T) {
	uc := application.NewCreateEventUseCase(newFakeEventRepo())

	_, err := uc.Execute(context.Background(), application.CreateEventInput{
		OrganizerID: "org-1",
		Name:        "Old Event",
		Date:        "2020-01-01T00:00:00Z",
		Location:    "Jakarta",
	})

	if err != domain.ErrDateMustBeFuture {
		t.Fatalf("expected ErrDateMustBeFuture, got %v", err)
	}
}

func TestCreateEventUseCase_Execute_MissingFields(t *testing.T) {
	uc := application.NewCreateEventUseCase(newFakeEventRepo())

	_, err := uc.Execute(context.Background(), application.CreateEventInput{
		OrganizerID: "org-1",
		Name:        "",
		Date:        futureDate,
		Location:    "Jakarta",
	})

	if err != domain.ErrNameRequired {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}
}

func TestCreateTicketTypeUseCase_Execute_Success(t *testing.T) {
	repo := newFakeEventRepo()
	// Seed event directly into map to bypass ID generation in fake
	repo.events["event-1"] = &domain.Event{
		ID: "event-1", Name: "Test", Location: "JKT",
		Date: time.Now().Add(24 * time.Hour),
	}

	uc := application.NewCreateTicketTypeUseCase(repo)
	tt, err := uc.Execute(context.Background(), application.CreateTicketTypeInput{
		EventID:    "event-1",
		Name:       "Festival GA",
		Price:      150000,
		Kind:       "GA",
		TotalQuota: 10000,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tt.TotalQuota != 10000 {
		t.Errorf("total_quota = %d, want 10000", tt.TotalQuota)
	}
}

func TestCreateTicketTypeUseCase_Execute_EventNotFound(t *testing.T) {
	uc := application.NewCreateTicketTypeUseCase(newFakeEventRepo())

	_, err := uc.Execute(context.Background(), application.CreateTicketTypeInput{
		EventID: "nonexistent", Name: "GA", Price: 0, Kind: "GA", TotalQuota: 100,
	})

	if err != domain.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}
}

func TestProvisionUnitsUseCase_Execute_Success(t *testing.T) {
	repo := newFakeEventRepo()
	repo.ticketTypes["tt-1"] = &domain.TicketType{
		ID: "tt-1", TotalQuota: 100, Kind: domain.KindGA,
	}
	uc := application.NewProvisionUnitsUseCase(repo)

	out, err := uc.Execute(context.Background(), application.ProvisionUnitsInput{
		TicketTypeID: "tt-1",
		Quantity:     50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Provisioned != 50 {
		t.Errorf("provisioned = %d, want 50", out.Provisioned)
	}
	if repo.unitCounts["tt-1"] != 50 {
		t.Errorf("unit count = %d, want 50", repo.unitCounts["tt-1"])
	}
}

func TestProvisionUnitsUseCase_Execute_QuotaExceeded(t *testing.T) {
	repo := newFakeEventRepo()
	repo.ticketTypes["tt-1"] = &domain.TicketType{
		ID: "tt-1", TotalQuota: 100, Kind: domain.KindGA,
	}
	repo.unitCounts["tt-1"] = 90 // sudah 90 dari 100

	uc := application.NewProvisionUnitsUseCase(repo)
	_, err := uc.Execute(context.Background(), application.ProvisionUnitsInput{
		TicketTypeID: "tt-1",
		Quantity:     20, // minta 20, sisa cuma 10
	})

	if err == nil {
		t.Fatal("expected error karena quota exceeded")
	}
}

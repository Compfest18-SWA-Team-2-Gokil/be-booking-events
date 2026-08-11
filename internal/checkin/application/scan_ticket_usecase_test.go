package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/checkin/application"
	"github.com/ebk-tech/be-booking-events/internal/checkin/domain"
)

func TestScanTicketUseCase_Execute_Success(t *testing.T) {
	payload := &domain.QRPayload{
		TicketUnitID: "unit-1",
		OrderID:      "order-1",
		EventID:      "event-1",
		IssuedAt:     time.Now(),
	}

	repo := newFakeCheckinRepo()
	repo.units["unit-1"] = &domain.CheckinTicket{ID: "unit-1", OrderID: "order-1", EventID: "event-1"}

	signer := &fakeSigner{validContent: "valid.qr", payload: payload}
	uc := application.NewScanTicketUseCase(repo, signer)

	out, err := uc.Execute(context.Background(), application.ScanTicketInput{
		QRContent:      "valid.qr",
		GateOperatorID: "gate-op-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TicketUnitID != "unit-1" {
		t.Errorf("ticket_unit_id = %s, want unit-1", out.TicketUnitID)
	}
	if out.EventID != "event-1" {
		t.Errorf("event_id = %s, want event-1", out.EventID)
	}
	if !repo.admittedIDs["unit-1"] {
		t.Error("unit-1 seharusnya sudah ter-admit")
	}
}

func TestScanTicketUseCase_Execute_InvalidSignature(t *testing.T) {
	signer := &fakeSigner{validContent: "valid.qr", verifyErr: domain.ErrInvalidSignature}
	uc := application.NewScanTicketUseCase(newFakeCheckinRepo(), signer)

	_, err := uc.Execute(context.Background(), application.ScanTicketInput{
		QRContent: "tampered.qr",
	})

	if err != domain.ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestScanTicketUseCase_Execute_WrongQRContent(t *testing.T) {
	signer := &fakeSigner{validContent: "valid.qr", payload: &domain.QRPayload{
		TicketUnitID: "unit-1", EventID: "event-1", IssuedAt: time.Now(),
	}}
	uc := application.NewScanTicketUseCase(newFakeCheckinRepo(), signer)

	_, err := uc.Execute(context.Background(), application.ScanTicketInput{
		QRContent: "wrong.qr",
	})

	if err != domain.ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestScanTicketUseCase_Execute_AlreadyAdmitted(t *testing.T) {
	payload := &domain.QRPayload{
		TicketUnitID: "unit-1", EventID: "event-1", IssuedAt: time.Now(),
	}

	repo := newFakeCheckinRepo()
	repo.units["unit-1"] = &domain.CheckinTicket{ID: "unit-1", EventID: "event-1"}
	repo.admittedIDs["unit-1"] = true // sudah admitted sebelumnya

	signer := &fakeSigner{validContent: "valid.qr", payload: payload}
	uc := application.NewScanTicketUseCase(repo, signer)

	_, err := uc.Execute(context.Background(), application.ScanTicketInput{
		QRContent: "valid.qr",
	})

	if err != domain.ErrAlreadyAdmitted {
		t.Fatalf("expected ErrAlreadyAdmitted, got %v", err)
	}
}

func TestScanTicketUseCase_Execute_InvalidPayload(t *testing.T) {
	// Payload dengan TicketUnitID kosong → Validate() gagal
	payload := &domain.QRPayload{EventID: "event-1", IssuedAt: time.Now()}
	signer := &fakeSigner{validContent: "valid.qr", payload: payload}
	uc := application.NewScanTicketUseCase(newFakeCheckinRepo(), signer)

	_, err := uc.Execute(context.Background(), application.ScanTicketInput{
		QRContent: "valid.qr",
	})

	if err != domain.ErrInvalidQRPayload {
		t.Fatalf("expected ErrInvalidQRPayload, got %v", err)
	}
}

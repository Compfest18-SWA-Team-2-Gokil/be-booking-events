package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/domain"
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
	repo.assignments["gate-op-1:event-1"] = true // gate operator di-assign ke event ini

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

func TestScanTicketUseCase_Execute_GateOperatorNotAssigned(t *testing.T) {
	payload := &domain.QRPayload{
		TicketUnitID: "unit-1", EventID: "event-1", IssuedAt: time.Now(),
	}
	repo := newFakeCheckinRepo()
	// assignments kosong → gate operator belum di-assign
	signer := &fakeSigner{validContent: "valid.qr", payload: payload}
	uc := application.NewScanTicketUseCase(repo, signer)

	_, err := uc.Execute(context.Background(), application.ScanTicketInput{
		QRContent: "valid.qr", GateOperatorID: "unassigned-gate-op",
	})
	if err != domain.ErrGateOperatorNotAssigned {
		t.Fatalf("expected ErrGateOperatorNotAssigned, got %v", err)
	}
}

func TestScanTicketUseCase_Execute_AlreadyAdmitted(t *testing.T) {
	payload := &domain.QRPayload{
		TicketUnitID: "unit-1", EventID: "event-1", IssuedAt: time.Now(),
	}

	repo := newFakeCheckinRepo()
	repo.units["unit-1"] = &domain.CheckinTicket{ID: "unit-1", EventID: "event-1"}
	repo.admittedIDs["unit-1"] = true            // sudah admitted sebelumnya
	repo.assignments["gate-op-1:event-1"] = true // gate operator valid

	signer := &fakeSigner{validContent: "valid.qr", payload: payload}
	uc := application.NewScanTicketUseCase(repo, signer)

	_, err := uc.Execute(context.Background(), application.ScanTicketInput{
		QRContent: "valid.qr", GateOperatorID: "gate-op-1",
	})

	if err != domain.ErrAlreadyAdmitted {
		t.Fatalf("expected ErrAlreadyAdmitted, got %v", err)
	}
}

func TestScanTicketUseCase_Execute_InvalidPayload(t *testing.T) {
	// Payload dengan TicketUnitID kosong → Validate() gagal (sebelum cek assignment)
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

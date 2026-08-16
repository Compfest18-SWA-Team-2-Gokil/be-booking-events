package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/domain"
)

type ScanTicketUseCase struct {
	repo   CheckinRepository
	signer QRSigner
}

func NewScanTicketUseCase(repo CheckinRepository, signer QRSigner) *ScanTicketUseCase {
	return &ScanTicketUseCase{repo: repo, signer: signer}
}

type ScanTicketInput struct {
	QRContent      string
	GateOperatorID string
}

type ScanTicketOutput struct {
	TicketUnitID string `json:"ticket_unit_id"`
	EventID      string `json:"event_id"`
}

func (uc *ScanTicketUseCase) Execute(ctx context.Context, input ScanTicketInput) (*ScanTicketOutput, error) {
	payload, err := uc.signer.Verify(input.QRContent)
	if err != nil {
		return nil, domain.ErrInvalidSignature
	}

	if err := payload.Validate(); err != nil {
		return nil, err
	}

	// RBAC: pastikan gate operator di-assign ke event dari tiket ini.
	assigned, err := uc.repo.IsGateOperatorAssigned(ctx, input.GateOperatorID, payload.EventID)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, domain.ErrGateOperatorNotAssigned
	}

	// Atomic UPDATE: CONFIRMED → ADMITTED. Jika 0 baris → sudah dipakai atau bukan CONFIRMED.
	if err := uc.repo.AdmitUnit(ctx, payload.TicketUnitID, input.GateOperatorID); err != nil {
		return nil, err
	}

	return &ScanTicketOutput{
		TicketUnitID: payload.TicketUnitID,
		EventID:      payload.EventID,
	}, nil
}

package application

import (
	"context"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/domain"
)

type IssueTicketUseCase struct {
	repo   CheckinRepository
	signer QRSigner
}

func NewIssueTicketUseCase(repo CheckinRepository, signer QRSigner) *IssueTicketUseCase {
	return &IssueTicketUseCase{repo: repo, signer: signer}
}

type IssueTicketOutput struct {
	QRContent string `json:"qr_content"`
}

func (uc *IssueTicketUseCase) Execute(ctx context.Context, ticketUnitID string) (*IssueTicketOutput, error) {
	ticket, err := uc.repo.GetConfirmedUnit(ctx, ticketUnitID)
	if err != nil {
		return nil, err
	}

	payload := domain.QRPayload{
		TicketUnitID: ticket.ID,
		OrderID:      ticket.OrderID,
		EventID:      ticket.EventID,
		IssuedAt:     time.Now().UTC(),
	}

	qr, err := uc.signer.Sign(payload)
	if err != nil {
		return nil, err
	}

	return &IssueTicketOutput{QRContent: qr}, nil
}

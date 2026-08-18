package application

import (
	"context"
	"errors"
)

var (
	ErrTargetOrderRequired = errors.New("target_order_id wajib diisi")
	ErrReasonRequired      = errors.New("reason wajib diisi untuk audit log")
)

type ReassignTicketUseCase struct {
	repo AdminRepository
}

func NewReassignTicketUseCase(repo AdminRepository) *ReassignTicketUseCase {
	return &ReassignTicketUseCase{repo: repo}
}

type ReassignTicketInput struct {
	UnitID        string
	AdminID       string
	TargetOrderID string
	NewSeatNumber string
	Reason        string
}

func (uc *ReassignTicketUseCase) Execute(ctx context.Context, input ReassignTicketInput) error {
	if input.TargetOrderID == "" {
		return ErrTargetOrderRequired
	}
	if input.Reason == "" {
		return ErrReasonRequired
	}

	return uc.repo.ReassignTicket(ctx, input.UnitID, input.AdminID, input.TargetOrderID, input.NewSeatNumber, input.Reason)
}

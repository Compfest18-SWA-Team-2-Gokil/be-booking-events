package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/auth/domain"
)

type AssignGateOperatorUseCase struct {
	repo UserRepository
}

func NewAssignGateOperatorUseCase(repo UserRepository) *AssignGateOperatorUseCase {
	return &AssignGateOperatorUseCase{repo: repo}
}

type AssignGateOperatorInput struct {
	GateOperatorUserID string
	EventID            string
}

func (uc *AssignGateOperatorUseCase) Execute(ctx context.Context, input AssignGateOperatorInput) error {
	user, err := uc.repo.FindByID(ctx, input.GateOperatorUserID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	if user.Role != domain.RoleGateOperator {
		return domain.ErrNotGateOperator
	}

	return uc.repo.AssignGateOperator(ctx, input.GateOperatorUserID, input.EventID)
}

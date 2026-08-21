package application

import (
	"context"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type AssignGateOperatorUseCase struct {
	repo      UserRepository
	eventRepo EventOwnershipChecker
}

func NewAssignGateOperatorUseCase(repo UserRepository, eventRepo EventOwnershipChecker) *AssignGateOperatorUseCase {
	return &AssignGateOperatorUseCase{repo: repo, eventRepo: eventRepo}
}

type AssignGateOperatorInput struct {
	Username   string
	EventID    string
	OrganizerID string
}

type AssignedOperatorOutput struct {
	Status     string    `json:"status"`
	User       UserBrief `json:"user"`
	AssignedAt time.Time `json:"assigned_at"`
}

type UserBrief struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func (uc *AssignGateOperatorUseCase) Execute(ctx context.Context, input AssignGateOperatorInput) (*AssignedOperatorOutput, error) {
	orgID, err := uc.eventRepo.GetEventOrganizerID(ctx, input.EventID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}
	if orgID != input.OrganizerID {
		return nil, domain.ErrNotEventOrganizer
	}

	user, err := uc.repo.FindByUsername(ctx, input.Username)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if user.Role != domain.RoleGateOperator {
		return nil, domain.ErrNotGateOperator
	}

	assigned := time.Now()
	if err := uc.repo.AssignGateOperator(ctx, user.ID, input.EventID, input.OrganizerID); err != nil {
		return nil, err
	}

	return &AssignedOperatorOutput{
		Status: "ACTIVE",
		User: UserBrief{
			ID:       user.ID,
			Username: user.Username,
			Name:     user.Name,
			Email:    user.Email,
		},
		AssignedAt: assigned,
	}, nil
}

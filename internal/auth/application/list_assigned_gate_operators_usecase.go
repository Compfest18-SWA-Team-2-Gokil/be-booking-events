package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type ListAssignedGateOperatorsUseCase struct {
	repo      UserRepository
	eventRepo EventOwnershipChecker
}

func NewListAssignedGateOperatorsUseCase(repo UserRepository, eventRepo EventOwnershipChecker) *ListAssignedGateOperatorsUseCase {
	return &ListAssignedGateOperatorsUseCase{repo: repo, eventRepo: eventRepo}
}

func (uc *ListAssignedGateOperatorsUseCase) Execute(ctx context.Context, eventID, organizerID string) ([]AssignedOperator, error) {
	orgID, err := uc.eventRepo.GetEventOrganizerID(ctx, eventID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}
	if orgID != organizerID {
		return nil, domain.ErrNotEventOrganizer
	}

	return uc.repo.ListAssignedGateOperators(ctx, eventID)
}

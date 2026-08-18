package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type RemoveGateOperatorUseCase struct {
	repo      UserRepository
	eventRepo EventOwnershipChecker
}

func NewRemoveGateOperatorUseCase(repo UserRepository, eventRepo EventOwnershipChecker) *RemoveGateOperatorUseCase {
	return &RemoveGateOperatorUseCase{repo: repo, eventRepo: eventRepo}
}

func (uc *RemoveGateOperatorUseCase) Execute(ctx context.Context, userID, eventID, organizerID string) error {
	orgID, err := uc.eventRepo.GetEventOrganizerID(ctx, eventID)
	if err != nil {
		return domain.ErrUserNotFound
	}
	if orgID != organizerID {
		return domain.ErrNotEventOrganizer
	}

	return uc.repo.RemoveGateOperator(ctx, userID, eventID)
}

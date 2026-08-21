package application

import (
	"context"
	"fmt"
)

type ListMyAssignedEventsUseCase struct {
	userRepo UserRepository
}

func NewListMyAssignedEventsUseCase(userRepo UserRepository) *ListMyAssignedEventsUseCase {
	return &ListMyAssignedEventsUseCase{
		userRepo: userRepo,
	}
}

func (uc *ListMyAssignedEventsUseCase) Execute(ctx context.Context, userID string) ([]AssignedEvent, error) {
	events, err := uc.userRepo.ListMyAssignedEvents(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list my assigned events: %w", err)
	}
	if events == nil {
		events = []AssignedEvent{}
	}
	return events, nil
}

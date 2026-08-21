package application

import (
	"context"
)

type CheckUsernameAvailabilityUseCase struct {
	repo UserRepository
}

func NewCheckUsernameAvailabilityUseCase(repo UserRepository) *CheckUsernameAvailabilityUseCase {
	return &CheckUsernameAvailabilityUseCase{repo: repo}
}

type UsernameAvailabilityResult struct {
	Available bool `json:"available"`
}

func (uc *CheckUsernameAvailabilityUseCase) Execute(ctx context.Context, username, currentUserID string) (*UsernameAvailabilityResult, error) {
	existing, _ := uc.repo.FindByUsername(ctx, username)
	if existing == nil {
		return &UsernameAvailabilityResult{Available: true}, nil
	}
	if existing.ID == currentUserID {
		return &UsernameAvailabilityResult{Available: true}, nil
	}
	return &UsernameAvailabilityResult{Available: false}, nil
}

package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type UpdateUsernameUseCase struct {
	repo UserRepository
}

func NewUpdateUsernameUseCase(repo UserRepository) *UpdateUsernameUseCase {
	return &UpdateUsernameUseCase{repo: repo}
}

func (uc *UpdateUsernameUseCase) Execute(ctx context.Context, userID, newUsername string) (*domain.User, error) {
	if !domain.UsernameRegex.MatchString(newUsername) {
		return nil, domain.ErrInvalidUsername
	}

	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Username == newUsername {
		return nil, domain.ErrSameUsername
	}

	existing, _ := uc.repo.FindByUsername(ctx, newUsername)
	if existing != nil {
		return nil, domain.ErrUsernameAlreadyTaken
	}

	if err := uc.repo.UpdateUsername(ctx, userID, newUsername); err != nil {
		return nil, err
	}

	user.Username = newUsername
	return user, nil
}

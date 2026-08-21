package application

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type ChangePasswordUseCase struct {
	repo   UserRepository
	hasher PasswordHasher
}

func NewChangePasswordUseCase(repo UserRepository, hasher PasswordHasher) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{repo: repo, hasher: hasher}
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, userID, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return domain.ErrNewPasswordTooShort
	}

	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := uc.hasher.Verify(user.PasswordHash, oldPassword); err != nil {
		return domain.ErrWrongPassword
	}

	hash, err := uc.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return uc.repo.UpdatePassword(ctx, userID, hash)
}

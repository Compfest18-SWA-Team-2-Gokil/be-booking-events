package application

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type RegisterUseCase struct {
	repo   UserRepository
	hasher PasswordHasher
}

func NewRegisterUseCase(repo UserRepository, hasher PasswordHasher) *RegisterUseCase {
	return &RegisterUseCase{repo: repo, hasher: hasher}
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
	Role     domain.Role
}

func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (*domain.User, error) {
	if len(input.Password) < 8 {
		return nil, domain.ErrPasswordTooShort
	}

	if input.Role == "" {
		input.Role = domain.RoleBuyer
	}

	user := &domain.User{
		Email: input.Email,
		Name:  input.Name,
		Role:  input.Role,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}

	existing, _ := uc.repo.FindByEmail(ctx, input.Email)
	if existing != nil {
		return nil, domain.ErrEmailAlreadyTaken
	}

	hash, err := uc.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = hash

	if err := uc.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

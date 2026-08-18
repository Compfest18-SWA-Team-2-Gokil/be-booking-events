package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type LoginUseCase struct {
	repo          UserRepository
	hasher        PasswordHasher
	tokenProvider TokenProvider
}

func NewLoginUseCase(repo UserRepository, hasher PasswordHasher, tokenProvider TokenProvider) *LoginUseCase {
	return &LoginUseCase{repo: repo, hasher: hasher, tokenProvider: tokenProvider}
}

type LoginOutput struct {
	Token string
	User  *domain.User
}

func (uc *LoginUseCase) Execute(ctx context.Context, email, password string) (*LoginOutput, error) {
	user, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		// Sembunyikan detail error agar tidak bocorkan apakah email terdaftar.
		return nil, domain.ErrInvalidCredentials
	}

	if err := uc.hasher.Verify(user.PasswordHash, password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := uc.tokenProvider.Generate(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &LoginOutput{Token: token, User: user}, nil
}

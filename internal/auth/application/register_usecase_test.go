package application_test

import (
	"context"
	"testing"

	"github.com/ebk-tech/be-booking-events/internal/auth/application"
	"github.com/ebk-tech/be-booking-events/internal/auth/domain"
)

func TestRegisterUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	uc := application.NewRegisterUseCase(repo, &fakePasswordHasher{})

	user, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "buyer@test.com",
		Name:     "Budi Buyer",
		Password: "password123",
		Role:     domain.RoleBuyer,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "buyer@test.com" {
		t.Errorf("email = %s, want buyer@test.com", user.Email)
	}
	if user.PasswordHash != "hashed:password123" {
		t.Errorf("password tidak di-hash dengan benar")
	}
}

func TestRegisterUseCase_Execute_DefaultRoleBuyer(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	user, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "test@test.com",
		Name:     "Test User",
		Password: "password123",
		// Role tidak diisi → default ke BUYER
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RoleBuyer {
		t.Errorf("role = %s, want BUYER", user.Role)
	}
}

func TestRegisterUseCase_Execute_DuplicateEmail(t *testing.T) {
	repo := newFakeUserRepo()
	uc := application.NewRegisterUseCase(repo, &fakePasswordHasher{})

	input := application.RegisterInput{Email: "dup@test.com", Name: "A", Password: "password123", Role: domain.RoleBuyer}
	uc.Execute(context.Background(), input)

	_, err := uc.Execute(context.Background(), input)
	if err != domain.ErrEmailAlreadyTaken {
		t.Fatalf("expected ErrEmailAlreadyTaken, got %v", err)
	}
}

func TestRegisterUseCase_Execute_PasswordTooShort(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email: "a@b.com", Name: "A", Password: "short", Role: domain.RoleBuyer,
	})
	if err != domain.ErrPasswordTooShort {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegisterUseCase_Execute_InvalidEmail(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email: "bukan-email", Name: "A", Password: "password123", Role: domain.RoleBuyer,
	})
	if err != domain.ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

package application_test

import (
	"context"
	"testing"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

func TestRegisterUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	uc := application.NewRegisterUseCase(repo, &fakePasswordHasher{})

	user, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "buyer@test.com",
		Username: "buyer_test",
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
	if user.Username != "buyer_test" {
		t.Errorf("username = %s, want buyer_test", user.Username)
	}
	if user.PasswordHash != "hashed:password123" {
		t.Errorf("password tidak di-hash dengan benar")
	}
}

func TestRegisterUseCase_Execute_DefaultRoleBuyer(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	user, err := uc.Execute(context.Background(), application.RegisterInput{
		Email:    "test@test.com",
		Username: "testuser",
		Name:     "Test User",
		Password: "password123",
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

	input := application.RegisterInput{Email: "dup@test.com", Username: "dup1", Name: "A", Password: "password123", Role: domain.RoleBuyer}
	uc.Execute(context.Background(), input)

	_, err := uc.Execute(context.Background(), application.RegisterInput{Email: "dup@test.com", Username: "dup2", Name: "A2", Password: "password123", Role: domain.RoleBuyer})
	if err != domain.ErrEmailAlreadyTaken {
		t.Fatalf("expected ErrEmailAlreadyTaken, got %v", err)
	}
}

func TestRegisterUseCase_Execute_DuplicateUsername(t *testing.T) {
	repo := newFakeUserRepo()
	uc := application.NewRegisterUseCase(repo, &fakePasswordHasher{})

	input := application.RegisterInput{Email: "first@test.com", Username: "sameuser", Name: "A", Password: "password123", Role: domain.RoleBuyer}
	uc.Execute(context.Background(), input)

	_, err := uc.Execute(context.Background(), application.RegisterInput{Email: "second@test.com", Username: "sameuser", Name: "B", Password: "password123", Role: domain.RoleBuyer})
	if err != domain.ErrUsernameAlreadyTaken {
		t.Fatalf("expected ErrUsernameAlreadyTaken, got %v", err)
	}
}

func TestRegisterUseCase_Execute_InvalidUsername(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email: "a@b.com", Username: "AB", Name: "A", Password: "password123", Role: domain.RoleBuyer,
	})
	if err != domain.ErrInvalidUsername {
		t.Fatalf("expected ErrInvalidUsername, got %v", err)
	}
}

func TestRegisterUseCase_Execute_UsernameTooShort(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email: "a@b.com", Username: "ab", Name: "A", Password: "password123", Role: domain.RoleBuyer,
	})
	if err != domain.ErrInvalidUsername {
		t.Fatalf("expected ErrInvalidUsername, got %v", err)
	}
}

func TestRegisterUseCase_Execute_PasswordTooShort(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email: "a@b.com", Username: "validuser", Name: "A", Password: "short", Role: domain.RoleBuyer,
	})
	if err != domain.ErrPasswordTooShort {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegisterUseCase_Execute_InvalidEmail(t *testing.T) {
	uc := application.NewRegisterUseCase(newFakeUserRepo(), &fakePasswordHasher{})

	_, err := uc.Execute(context.Background(), application.RegisterInput{
		Email: "bukan-email", Username: "validuser", Name: "A", Password: "password123", Role: domain.RoleBuyer,
	})
	if err != domain.ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

package application_test

import (
	"context"
	"testing"

	"github.com/ebk-tech/be-booking-events/internal/auth/application"
	"github.com/ebk-tech/be-booking-events/internal/auth/domain"
)

func seedUser(repo *fakeUserRepo, email, name string, role domain.Role) {
	repo.Create(context.Background(), &domain.User{
		Email:        email,
		Name:         name,
		Role:         role,
		PasswordHash: "hashed:password123",
	})
}

func TestLoginUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "organizer@test.com", "Org", domain.RoleOrganizer)

	uc := application.NewLoginUseCase(repo, &fakePasswordHasher{}, &fakeTokenProvider{})

	out, err := uc.Execute(context.Background(), "organizer@test.com", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Token == "" {
		t.Error("token harus diisi")
	}
	if out.User.Role != domain.RoleOrganizer {
		t.Errorf("role = %s, want ORGANIZER", out.User.Role)
	}
}

func TestLoginUseCase_Execute_WrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "user@test.com", "User", domain.RoleBuyer)
	uc := application.NewLoginUseCase(repo, &fakePasswordHasher{}, &fakeTokenProvider{})

	_, err := uc.Execute(context.Background(), "user@test.com", "wrongpassword")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUseCase_Execute_EmailNotFound(t *testing.T) {
	uc := application.NewLoginUseCase(newFakeUserRepo(), &fakePasswordHasher{}, &fakeTokenProvider{})

	_, err := uc.Execute(context.Background(), "notfound@test.com", "password123")
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials (bukan ErrUserNotFound), got %v", err)
	}
}

func TestAssignGateOperatorUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "gate@test.com", "Gate Op", domain.RoleGateOperator)
	gateUser, _ := repo.FindByEmail(context.Background(), "gate@test.com")

	uc := application.NewAssignGateOperatorUseCase(repo)
	err := uc.Execute(context.Background(), application.AssignGateOperatorInput{
		GateOperatorUserID: gateUser.ID,
		EventID:            "event-123",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.assignments[gateUser.ID+":event-123"] {
		t.Error("assignment seharusnya tersimpan")
	}
}

func TestAssignGateOperatorUseCase_Execute_NotGateOperator(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "buyer@test.com", "Buyer", domain.RoleBuyer)
	buyer, _ := repo.FindByEmail(context.Background(), "buyer@test.com")

	uc := application.NewAssignGateOperatorUseCase(repo)
	err := uc.Execute(context.Background(), application.AssignGateOperatorInput{
		GateOperatorUserID: buyer.ID,
		EventID:            "event-123",
	})

	if err != domain.ErrNotGateOperator {
		t.Fatalf("expected ErrNotGateOperator, got %v", err)
	}
}

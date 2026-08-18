package application_test

import (
	"context"
	"testing"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

func seedUser(repo *fakeUserRepo, email, username, name string, role domain.Role) {
	repo.Create(context.Background(), &domain.User{
		Email:        email,
		Username:     username,
		Name:         name,
		Role:         role,
		PasswordHash: "hashed:password123",
	})
}

func TestLoginUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "organizer@test.com", "organizer", "Org", domain.RoleOrganizer)

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
	seedUser(repo, "user@test.com", "user1", "User", domain.RoleBuyer)
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
	seedUser(repo, "gate@test.com", "gate_op1", "Gate Op", domain.RoleGateOperator)

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	uc := application.NewAssignGateOperatorUseCase(repo, eventChecker)
	result, err := uc.Execute(context.Background(), application.AssignGateOperatorInput{
		Username:    "gate_op1",
		EventID:     "event-123",
		OrganizerID: "org-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "assigned" {
		t.Errorf("status = %s, want assigned", result.Status)
	}
	if result.User.Username != "gate_op1" {
		t.Errorf("username = %s, want gate_op1", result.User.Username)
	}
}

func TestAssignGateOperatorUseCase_Execute_NotGateOperator(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "buyer@test.com", "buyer1", "Buyer", domain.RoleBuyer)

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	uc := application.NewAssignGateOperatorUseCase(repo, eventChecker)
	_, err := uc.Execute(context.Background(), application.AssignGateOperatorInput{
		Username:    "buyer1",
		EventID:     "event-123",
		OrganizerID: "org-1",
	})

	if err != domain.ErrNotGateOperator {
		t.Fatalf("expected ErrNotGateOperator, got %v", err)
	}
}

func TestAssignGateOperatorUseCase_Execute_NotEventOrganizer(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "gate@test.com", "gate_op2", "Gate Op", domain.RoleGateOperator)

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	uc := application.NewAssignGateOperatorUseCase(repo, eventChecker)
	_, err := uc.Execute(context.Background(), application.AssignGateOperatorInput{
		Username:    "gate_op2",
		EventID:     "event-123",
		OrganizerID: "org-wrong",
	})

	if err != domain.ErrNotEventOrganizer {
		t.Fatalf("expected ErrNotEventOrganizer, got %v", err)
	}
}

func TestAssignGateOperatorUseCase_Execute_UserNotFound(t *testing.T) {
	repo := newFakeUserRepo()

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	uc := application.NewAssignGateOperatorUseCase(repo, eventChecker)
	_, err := uc.Execute(context.Background(), application.AssignGateOperatorInput{
		Username:    "nonexistent",
		EventID:     "event-123",
		OrganizerID: "org-1",
	})

	if err != domain.ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestListAssignedGateOperatorsUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "gate@test.com", "gate_op3", "Gate Op", domain.RoleGateOperator)
	gateUser, _ := repo.FindByUsername(context.Background(), "gate_op3")

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	assignUC := application.NewAssignGateOperatorUseCase(repo, eventChecker)
	assignUC.Execute(context.Background(), application.AssignGateOperatorInput{
		Username:    "gate_op3",
		EventID:     "event-123",
		OrganizerID: "org-1",
	})

	listUC := application.NewListAssignedGateOperatorsUseCase(repo, eventChecker)
	operators, err := listUC.Execute(context.Background(), "event-123", "org-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(operators) != 1 {
		t.Fatalf("expected 1 operator, got %d", len(operators))
	}
	if operators[0].Username != "gate_op3" {
		t.Errorf("username = %s, want gate_op3", operators[0].Username)
	}
	_ = gateUser
}

func TestListAssignedGateOperatorsUseCase_Execute_NotOwner(t *testing.T) {
	repo := newFakeUserRepo()

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	listUC := application.NewListAssignedGateOperatorsUseCase(repo, eventChecker)
	_, err := listUC.Execute(context.Background(), "event-123", "org-wrong")

	if err != domain.ErrNotEventOrganizer {
		t.Fatalf("expected ErrNotEventOrganizer, got %v", err)
	}
}

func TestRemoveGateOperatorUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "gate@test.com", "gate_op4", "Gate Op", domain.RoleGateOperator)
	gateUser, _ := repo.FindByUsername(context.Background(), "gate_op4")

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	assignUC := application.NewAssignGateOperatorUseCase(repo, eventChecker)
	assignUC.Execute(context.Background(), application.AssignGateOperatorInput{
		Username:    "gate_op4",
		EventID:     "event-123",
		OrganizerID: "org-1",
	})

	removeUC := application.NewRemoveGateOperatorUseCase(repo, eventChecker)
	err := removeUC.Execute(context.Background(), gateUser.ID, "event-123", "org-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	operators, _ := repo.ListAssignedGateOperators(context.Background(), "event-123")
	for _, op := range operators {
		if op.UserID == gateUser.ID {
			t.Error("operator seharusnya sudah di-revoke")
		}
	}
}

func TestRemoveGateOperatorUseCase_Execute_NotOwner(t *testing.T) {
	repo := newFakeUserRepo()

	eventChecker := newFakeEventOwnershipChecker()
	eventChecker.organizerIDs["event-123"] = "org-1"

	removeUC := application.NewRemoveGateOperatorUseCase(repo, eventChecker)
	err := removeUC.Execute(context.Background(), "some-user-id", "event-123", "org-wrong")

	if err != domain.ErrNotEventOrganizer {
		t.Fatalf("expected ErrNotEventOrganizer, got %v", err)
	}
}

func TestSearchGateOperatorsUseCase_Execute_Success(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "gate1@test.com", "budi_sfo", "Budi Santoso", domain.RoleGateOperator)
	seedUser(repo, "gate2@test.com", "ani_gate", "Ani Gatekeeper", domain.RoleGateOperator)
	seedUser(repo, "buyer@test.com", "buyer99", "Buyer Biasa", domain.RoleBuyer)

	uc := application.NewSearchGateOperatorsUseCase(repo)
	users, err := uc.Execute(context.Background(), "budi")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Username != "budi_sfo" {
		t.Errorf("username = %s, want budi_sfo", users[0].Username)
	}
}

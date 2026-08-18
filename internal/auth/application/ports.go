package application

import (
	"context"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	AssignGateOperator(ctx context.Context, userID, eventID, assignedBy string) error
	ListAssignedGateOperators(ctx context.Context, eventID string) ([]AssignedOperator, error)
	RemoveGateOperator(ctx context.Context, userID, eventID string) error
	SearchGateOperators(ctx context.Context, query string) ([]domain.User, error)
}

type AssignedOperator struct {
	UserID     string
	Username   string
	Name       string
	Email      string
	AssignedAt time.Time
	AssignedBy string
	Status     string
}

type EventOwnershipChecker interface {
	GetEventOrganizerID(ctx context.Context, eventID string) (string, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) error
}

type TokenProvider interface {
	Generate(userID, role string) (string, error)
	Verify(token string) (userID, role string, err error)
}

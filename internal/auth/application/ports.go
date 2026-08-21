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
	UpdateUsername(ctx context.Context, userID, newUsername string) error
	UpdatePassword(ctx context.Context, userID, newPasswordHash string) error
	AssignGateOperator(ctx context.Context, userID, eventID, assignedBy string) error
	ListAssignedGateOperators(ctx context.Context, eventID string) ([]AssignedOperator, error)
	RemoveGateOperator(ctx context.Context, userID, eventID string) error
	SearchGateOperators(ctx context.Context, query string) ([]domain.User, error)
	ListMyAssignedEvents(ctx context.Context, userID string) ([]AssignedEvent, error)
}

type AssignedOperator struct {
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	AssignedAt time.Time `json:"assigned_at"`
	AssignedBy string    `json:"assigned_by"`
	Status     string    `json:"status"`
}

type AssignedEvent struct {
	EventID     string    `json:"event_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Date        time.Time `json:"date"`
	Location    string    `json:"location"`
	ImageURL    string    `json:"image_url,omitempty"`
	AssignedAt  time.Time `json:"assigned_at"`
	Status      string    `json:"status"`
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

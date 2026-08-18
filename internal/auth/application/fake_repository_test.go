package application_test

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type fakeUserRepo struct {
	byEmail     map[string]*domain.User
	byID        map[string]*domain.User
	byUsername  map[string]*domain.User
	assignments map[string]*fakeAssignment
}

type fakeAssignment struct {
	userID     string
	eventID    string
	assignedBy string
	status     string
	assignedAt time.Time
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byEmail:     make(map[string]*domain.User),
		byID:        make(map[string]*domain.User),
		byUsername:  make(map[string]*domain.User),
		assignments: make(map[string]*fakeAssignment),
	}
}

func (r *fakeUserRepo) Create(_ context.Context, user *domain.User) error {
	user.ID = "id-" + user.Email
	r.byEmail[user.Email] = user
	r.byID[user.ID] = user
	r.byUsername[user.Username] = user
	return nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) FindByUsername(_ context.Context, username string) (*domain.User, error) {
	u, ok := r.byUsername[username]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) AssignGateOperator(_ context.Context, userID, eventID, assignedBy string) error {
	key := userID + ":" + eventID
	if existing, ok := r.assignments[key]; ok && existing.status == "ACTIVE" {
		return domain.ErrAlreadyAssigned
	}
	r.assignments[key] = &fakeAssignment{
		userID:     userID,
		eventID:    eventID,
		assignedBy: assignedBy,
		status:     "ACTIVE",
		assignedAt: time.Now(),
	}
	return nil
}

func (r *fakeUserRepo) ListAssignedGateOperators(_ context.Context, eventID string) ([]application.AssignedOperator, error) {
	var operators []application.AssignedOperator
	for _, a := range r.assignments {
		if a.eventID == eventID && a.status == "ACTIVE" {
			u, ok := r.byID[a.userID]
			if ok {
				operators = append(operators, application.AssignedOperator{
					UserID:     a.userID,
					Username:   u.Username,
					Name:       u.Name,
					Email:      u.Email,
					AssignedAt: a.assignedAt,
					AssignedBy: a.assignedBy,
					Status:     a.status,
				})
			}
		}
	}
	return operators, nil
}

func (r *fakeUserRepo) RemoveGateOperator(_ context.Context, userID, eventID string) error {
	key := userID + ":" + eventID
	a, ok := r.assignments[key]
	if !ok || a.status != "ACTIVE" {
		return domain.ErrUserNotFound
	}
	a.status = "REVOKED"
	return nil
}

func (r *fakeUserRepo) SearchGateOperators(_ context.Context, query string) ([]domain.User, error) {
	var users []domain.User
	lowerQuery := strings.ToLower(query)
	for _, u := range r.byEmail {
		if u.Role == domain.RoleGateOperator {
			if strings.Contains(strings.ToLower(u.Username), lowerQuery) ||
				strings.Contains(strings.ToLower(u.Name), lowerQuery) {
				users = append(users, *u)
			}
		}
	}
	return users, nil
}

type fakeEventOwnershipChecker struct {
	organizerIDs map[string]string
}

func newFakeEventOwnershipChecker() *fakeEventOwnershipChecker {
	return &fakeEventOwnershipChecker{
		organizerIDs: make(map[string]string),
	}
}

func (f *fakeEventOwnershipChecker) GetEventOrganizerID(_ context.Context, eventID string) (string, error) {
	id, ok := f.organizerIDs[eventID]
	if !ok {
		return "", domain.ErrUserNotFound
	}
	return id, nil
}

type fakePasswordHasher struct{}

func (f *fakePasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (f *fakePasswordHasher) Verify(hash, password string) error {
	if hash != "hashed:"+password {
		return errors.New("password salah")
	}
	return nil
}

type fakeTokenProvider struct{}

func (f *fakeTokenProvider) Generate(userID, _ string) (string, error) {
	return "token:" + userID, nil
}

func (f *fakeTokenProvider) Verify(token string) (string, string, error) {
	if !strings.HasPrefix(token, "token:") {
		return "", "", domain.ErrInvalidToken
	}
	return strings.TrimPrefix(token, "token:"), "", nil
}

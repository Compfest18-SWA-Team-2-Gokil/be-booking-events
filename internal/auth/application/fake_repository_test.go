package application_test

import (
	"context"
	"errors"
	"strings"

	"github.com/ebk-tech/be-booking-events/internal/auth/domain"
)

type fakeUserRepo struct {
	byEmail     map[string]*domain.User
	byID        map[string]*domain.User
	assignments map[string]bool // "userID:eventID"
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byEmail:     make(map[string]*domain.User),
		byID:        make(map[string]*domain.User),
		assignments: make(map[string]bool),
	}
}

func (r *fakeUserRepo) Create(_ context.Context, user *domain.User) error {
	user.ID = "id-" + user.Email
	r.byEmail[user.Email] = user
	r.byID[user.ID] = user
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

func (r *fakeUserRepo) AssignGateOperator(_ context.Context, userID, eventID string) error {
	r.assignments[userID+":"+eventID] = true
	return nil
}

// fakePasswordHasher: hash = "hashed:" + password (deterministik, tanpa crypto).
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

// fakeTokenProvider: token = "token:" + userID.
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

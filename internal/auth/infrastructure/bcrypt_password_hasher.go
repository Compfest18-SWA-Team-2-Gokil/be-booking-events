package infrastructure

import (
	"github.com/ebk-tech/be-booking-events/internal/auth/application"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

type BcryptPasswordHasher struct{}

func NewBcryptPasswordHasher() *BcryptPasswordHasher { return &BcryptPasswordHasher{} }

var _ application.PasswordHasher = (*BcryptPasswordHasher)(nil)

func (h *BcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (h *BcryptPasswordHasher) Verify(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

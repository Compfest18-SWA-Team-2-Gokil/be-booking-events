package domain

import (
	"regexp"
	"strings"
)

var usernameRegex = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)

type Role string

const (
	RoleBuyer        Role = "BUYER"
	RoleOrganizer    Role = "ORGANIZER"
	RoleGateOperator Role = "GATE_OPERATOR"
	RoleAdmin        Role = "ADMIN"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleBuyer, RoleOrganizer, RoleGateOperator, RoleAdmin:
		return true
	}
	return false
}

type User struct {
	ID           string
	Email        string
	Username     string
	Name         string
	Role         Role
	PasswordHash string
}

func (u *User) Validate() error {
	if u.Username == "" {
		return ErrUsernameRequired
	}
	if !usernameRegex.MatchString(u.Username) {
		return ErrInvalidUsername
	}
	if !strings.Contains(u.Email, "@") {
		return ErrInvalidEmail
	}
	if u.Name == "" {
		return ErrNameRequired
	}
	if !u.Role.IsValid() {
		return ErrInvalidRole
	}
	return nil
}

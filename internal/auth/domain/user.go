package domain

import "strings"

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
	Name         string
	Role         Role
	PasswordHash string
}

func (u *User) Validate() error {
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

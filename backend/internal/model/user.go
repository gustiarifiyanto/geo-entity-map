package model

import (
	"slices"
	"strings"
	"time"
)

// Role decides what a user may do. It is the single source of truth for roles.
type Role string

const (
	// RoleUser can only view entities. Public registration always creates this role.
	RoleUser Role = "user"
	// RoleAdmin can also create, edit, move and delete entities.
	RoleAdmin Role = "admin"
)

// Roles lists every allowed role.
var Roles = []Role{RoleUser, RoleAdmin}

// Valid reports whether r is one of the allowed roles.
func (r Role) Valid() bool {
	return slices.Contains(Roles, r)
}

// User is an account. The password hash is never serialized.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// NormalizeEmail trims and lowercases an email so lookups are case-insensitive.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// RegisterInput is the request body for creating an account.
// The password is not trimmed: spaces are part of the password.
type RegisterInput struct {
	Email    string `json:"email" validate:"required,max=254,email"`
	Password string `json:"password" validate:"required,min=8,max_bytes=72"`
}

// Normalize prepares the input for validation. It must run before validation.
func (in *RegisterInput) Normalize() {
	in.Email = NormalizeEmail(in.Email)
}

// LoginInput is the request body for logging in. Only presence is checked,
// so the response never reveals the password rules of an existing account.
type LoginInput struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Normalize prepares the input for validation. It must run before validation.
func (in *LoginInput) Normalize() {
	in.Email = NormalizeEmail(in.Email)
}

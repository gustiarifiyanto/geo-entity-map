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

// UserStats counts accounts and their activity for the admin dashboard.
type UserStats struct {
	// Total, WithActiveSession and Online count only role user: admins are the
	// ones reading the dashboard, so counting them would blur the numbers.
	Total int `json:"total"`
	// ByRole has an entry for every role in Roles, including zero counts.
	ByRole map[Role]int `json:"by_role"`
	// WithActiveSession counts users (not sessions) with an unexpired session.
	WithActiveSession int `json:"with_active_session"`
	// Online counts users with an unexpired session seen within the online window.
	Online int `json:"online"`
	// ActiveUsers lists who is behind WithActiveSession (role user only), most
	// recently seen first, at most MaxActiveUsersListed.
	ActiveUsers []ActiveUser `json:"active_users"`
}

// MaxActiveUsersListed caps ActiveUsers so the dashboard response stays small.
const MaxActiveUsersListed = 50

// ActiveUser is an account with an unexpired session, for the admin dashboard.
type ActiveUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	// LastSeenAt is the latest activity over all of the user's sessions; nil
	// for sessions from before activity was recorded.
	LastSeenAt *time.Time `json:"last_seen_at"`
	Online     bool       `json:"online"`
}

// AdminStats is the response of GET /api/admin/stats.
type AdminStats struct {
	Users               UserStats `json:"users"`
	OnlineWindowMinutes int       `json:"online_window_minutes"`
}

// Demo accounts exist only when the server runs with DEMO_ACCOUNTS=true.
const (
	DemoAdminEmail = "admin@demo.local"
	DemoUserEmail  = "user@demo.local"
)

// DemoAccount is a ready-made login shown on the login page in demo mode.
// The password is generated at every server start and never stored in code.
type DemoAccount struct {
	Role     Role   `json:"role"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

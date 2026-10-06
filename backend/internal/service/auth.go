package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

const (
	// SessionTTL is how long a login lasts. The session cookie uses the same lifetime.
	SessionTTL = 7 * 24 * time.Hour
	// OnlineWindow is how recently a session must have been used for its user to count as online.
	OnlineWindow = 5 * time.Minute
	// lastSeenInterval limits how often a session's last_seen_at is written, so
	// most authenticated requests only read the database.
	lastSeenInterval = time.Minute
)

// UserRepository is the user and session storage the auth service depends on.
type UserRepository interface {
	CreateUser(ctx context.Context, u model.User) error
	UserByEmail(ctx context.Context, email string) (model.User, error)
	UpdatePasswordHash(ctx context.Context, id, hash string) error
	CreateSession(ctx context.Context, tokenHash, userID string, createdAt, expiresAt time.Time) error
	UserBySession(ctx context.Context, tokenHash string, now time.Time) (model.User, time.Time, error)
	TouchSession(ctx context.Context, tokenHash string, seenAt time.Time) error
	UserStats(ctx context.Context, role model.Role, now, onlineSince time.Time) (model.UserStats, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteExpiredSessions(ctx context.Context, now time.Time) error
}

// AuthService implements registration, login and sessions. Inputs passed to
// it must already have been validated and normalized.
type AuthService struct {
	repo UserRepository
	now  func() time.Time
	cost int
	// dummyHash is compared against when the email is unknown, so a login
	// takes about as long whether or not the account exists.
	dummyHash []byte
}

// NewAuthService returns a service backed by repo. bcryptCost is usually
// bcrypt.DefaultCost; tests pass bcrypt.MinCost to stay fast.
func NewAuthService(repo UserRepository, bcryptCost int) (*AuthService, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy password"), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash dummy password: %w", err)
	}
	return &AuthService{
		repo:      repo,
		now:       func() time.Time { return time.Now().UTC().Truncate(time.Second) },
		cost:      bcryptCost,
		dummyHash: dummy,
	}, nil
}

// Session is a logged-in user plus the raw token to put in the cookie.
type Session struct {
	User  model.User
	Token string
}

// Register creates an account with role user and logs it in, or returns
// model.ErrEmailTaken.
func (s *AuthService) Register(ctx context.Context, in model.RegisterInput) (Session, error) {
	u, err := s.createUser(ctx, in.Email, in.Password, model.RoleUser)
	if err != nil {
		return Session{}, err
	}
	return s.startSession(ctx, u)
}

// Login checks the credentials and starts a session, or returns
// model.ErrInvalidCredentials.
func (s *AuthService) Login(ctx context.Context, in model.LoginInput) (Session, error) {
	u, err := s.repo.UserByEmail(ctx, in.Email)
	if errors.Is(err, model.ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(in.Password))
		return Session{}, model.ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return Session{}, model.ErrInvalidCredentials
	}

	// Logins are rare, so this is a cheap place to keep the sessions table small.
	if err := s.repo.DeleteExpiredSessions(ctx, s.now()); err != nil {
		return Session{}, err
	}
	return s.startSession(ctx, u)
}

// Logout ends the session with the given token. Unknown tokens are ignored.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.repo.DeleteSession(ctx, hashToken(token))
}

// Authenticate returns the user of an unexpired session, or
// model.ErrUnauthenticated. It records the session as seen, at most once per
// lastSeenInterval.
func (s *AuthService) Authenticate(ctx context.Context, token string) (model.User, error) {
	if token == "" {
		return model.User{}, model.ErrUnauthenticated
	}
	now := s.now()
	tokenHash := hashToken(token)
	u, lastSeen, err := s.repo.UserBySession(ctx, tokenHash, now)
	if err != nil {
		return model.User{}, err
	}
	if now.Sub(lastSeen) >= lastSeenInterval {
		if err := s.repo.TouchSession(ctx, tokenHash, now); err != nil {
			return model.User{}, err
		}
	}
	return u, nil
}

// Stats counts users for the admin dashboard. Total, active-session and online
// counts cover role user only; ByRole lists every role, even with zero users.
func (s *AuthService) Stats(ctx context.Context) (model.AdminStats, error) {
	now := s.now()
	users, err := s.repo.UserStats(ctx, model.RoleUser, now, now.Add(-OnlineWindow))
	if err != nil {
		return model.AdminStats{}, err
	}
	for _, r := range model.Roles {
		if _, ok := users.ByRole[r]; !ok {
			users.ByRole[r] = 0
		}
	}
	return model.AdminStats{Users: users, OnlineWindowMinutes: int(OnlineWindow / time.Minute)}, nil
}

// EnsureAdmin creates an admin account unless the email is already registered.
// It reports the existing or new user and whether it was created.
func (s *AuthService) EnsureAdmin(ctx context.Context, in model.RegisterInput) (model.User, bool, error) {
	u, err := s.createUser(ctx, in.Email, in.Password, model.RoleAdmin)
	if errors.Is(err, model.ErrEmailTaken) {
		existing, err := s.repo.UserByEmail(ctx, in.Email)
		return existing, false, err
	}
	return u, err == nil, err
}

func (s *AuthService) createUser(ctx context.Context, email, password string, role model.Role) (model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return model.User{}, fmt.Errorf("hash password: %w", err)
	}
	u := model.User{
		ID:           uuid.NewString(),
		Email:        email,
		Role:         role,
		PasswordHash: string(hash),
		CreatedAt:    s.now(),
	}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (s *AuthService) startSession(ctx context.Context, u model.User) (Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Session{}, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	now := s.now()
	if err := s.repo.CreateSession(ctx, hashToken(token), u.ID, now, now.Add(SessionTTL)); err != nil {
		return Session{}, err
	}
	return Session{User: u, Token: token}, nil
}

// hashToken returns the value stored in the database for a cookie token.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// demoAccounts are the accounts SeedDemoAccounts keeps ready, one per role.
var demoAccounts = []struct {
	email string
	role  model.Role
}{
	{model.DemoAdminEmail, model.RoleAdmin},
	{model.DemoUserEmail, model.RoleUser},
}

// SeedDemoAccounts creates the demo admin and demo user, or resets their
// passwords, to a new random password each time it runs. It returns the
// logins to show on the login page. An email already registered with another
// role is left untouched and not returned, so demo mode never promotes anyone.
func (s *AuthService) SeedDemoAccounts(ctx context.Context) ([]model.DemoAccount, error) {
	accounts := make([]model.DemoAccount, 0, len(demoAccounts))
	for _, d := range demoAccounts {
		raw := make([]byte, 12)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("generate demo password: %w", err)
		}
		password := base64.RawURLEncoding.EncodeToString(raw)

		_, err := s.createUser(ctx, d.email, password, d.role)
		if errors.Is(err, model.ErrEmailTaken) {
			err = s.resetDemoPassword(ctx, d.email, d.role, password)
			if errors.Is(err, errDemoRoleTaken) {
				continue
			}
		}
		if err != nil {
			return nil, fmt.Errorf("seed demo account %s: %w", d.email, err)
		}
		accounts = append(accounts, model.DemoAccount{Role: d.role, Email: d.email, Password: password})
	}
	return accounts, nil
}

var errDemoRoleTaken = errors.New("demo email registered with another role")

func (s *AuthService) resetDemoPassword(ctx context.Context, email string, role model.Role, password string) error {
	u, err := s.repo.UserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if u.Role != role {
		return errDemoRoleTaken
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repo.UpdatePasswordHash(ctx, u.ID, string(hash))
}

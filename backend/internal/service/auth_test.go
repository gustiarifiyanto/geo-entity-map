package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/database"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
)

// clock is a settable time source for the service.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestService(t *testing.T) (*AuthService, *sql.DB, *clock) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := NewAuthService(repository.NewUserRepository(db), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	c := &clock{t: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)}
	s.now = c.now
	return s, db, c
}

func registerUser(t *testing.T, s *AuthService, email string) Session {
	t.Helper()
	sess, err := s.Register(context.Background(), model.RegisterInput{Email: email, Password: "rahasia123"})
	if err != nil {
		t.Fatalf("register %s: %v", email, err)
	}
	return sess
}

func lastSeen(t *testing.T, db *sql.DB, token string) time.Time {
	t.Helper()
	var raw string
	if err := db.QueryRow(`SELECT last_seen_at FROM sessions WHERE token_hash = ?`, hashToken(token)).Scan(&raw); err != nil {
		t.Fatalf("read last_seen_at: %v", err)
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t.Fatalf("parse last_seen_at %q: %v", raw, err)
	}
	return ts
}

func TestAuthenticateUpdatesLastSeenAtMostOncePerMinute(t *testing.T) {
	s, db, c := newTestService(t)
	ctx := context.Background()
	sess := registerUser(t, s, "budi@example.com")
	loginAt := c.t
	if got := lastSeen(t, db, sess.Token); !got.Equal(loginAt) {
		t.Fatalf("after login last_seen_at = %v, want %v", got, loginAt)
	}

	steps := []struct {
		advance time.Duration
		want    time.Time
	}{
		{30 * time.Second, loginAt},                  // too soon: not written
		{29 * time.Second, loginAt},                  // 59s after login: still not written
		{time.Second, loginAt.Add(time.Minute)},      // exactly 1 minute: written
		{10 * time.Second, loginAt.Add(time.Minute)}, // too soon again
		{3 * time.Minute, loginAt.Add(4*time.Minute + 10*time.Second)},
	}
	for i, st := range steps {
		c.advance(st.advance)
		if _, err := s.Authenticate(ctx, sess.Token); err != nil {
			t.Fatalf("step %d: Authenticate: %v", i, err)
		}
		if got := lastSeen(t, db, sess.Token); !got.Equal(st.want) {
			t.Errorf("step %d: last_seen_at = %v, want %v", i, got, st.want)
		}
	}
}

func TestAuthenticateSessionFromBeforeLastSeenColumn(t *testing.T) {
	s, db, _ := newTestService(t)
	sess := registerUser(t, s, "budi@example.com")
	if _, err := db.Exec(`UPDATE sessions SET last_seen_at = NULL`); err != nil {
		t.Fatalf("clear last_seen_at: %v", err)
	}
	if _, err := s.Authenticate(context.Background(), sess.Token); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	lastSeen(t, db, sess.Token) // fails if it is still NULL
}

func TestStats(t *testing.T) {
	s, db, c := newTestService(t)
	ctx := context.Background()

	if _, _, err := s.EnsureAdmin(ctx, model.RegisterInput{Email: "admin@example.com", Password: "admin-password"}); err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	// The admin has no session at all.

	// Two sessions for the same user: counted once.
	registerUser(t, s, "online@example.com")
	if _, err := s.Login(ctx, model.LoginInput{Email: "online@example.com", Password: "rahasia123"}); err != nil {
		t.Fatalf("second login: %v", err)
	}

	// Logged in 6 minutes ago and idle since: active but not online.
	c.advance(-6 * time.Minute)
	registerUser(t, s, "idle@example.com")
	c.advance(6 * time.Minute)

	// Expired session that was seen recently: neither active nor online.
	expired := registerUser(t, s, "expired@example.com")
	if _, err := db.Exec(`UPDATE sessions SET expires_at = ? WHERE token_hash = ?`,
		c.t.Add(-time.Second).Format(time.RFC3339), hashToken(expired.Token)); err != nil {
		t.Fatalf("expire session: %v", err)
	}

	// Seen exactly at the edge of the online window: online.
	c.advance(-OnlineWindow)
	registerUser(t, s, "edge@example.com")
	c.advance(OnlineWindow)

	got, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	want := model.UserStats{
		Total:             5,
		ByRole:            map[model.Role]int{model.RoleAdmin: 1, model.RoleUser: 4},
		WithActiveSession: 3, // online, idle, edge
		Online:            2, // online, edge
	}
	if got.Users.Total != want.Total || got.Users.WithActiveSession != want.WithActiveSession ||
		got.Users.Online != want.Online || len(got.Users.ByRole) != len(want.ByRole) ||
		got.Users.ByRole[model.RoleAdmin] != 1 || got.Users.ByRole[model.RoleUser] != 4 {
		t.Errorf("Stats().Users = %+v, want %+v", got.Users, want)
	}
	if got.OnlineWindowMinutes != 5 {
		t.Errorf("OnlineWindowMinutes = %d, want 5", got.OnlineWindowMinutes)
	}
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

const userColumns = `id, email, role, password_hash, created_at`

// UserRepository reads and writes users and their sessions in SQLite.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository returns a repository backed by db.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts u, or returns model.ErrEmailTaken if the email is already used.
func (r *UserRepository) CreateUser(ctx context.Context, u model.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (`+userColumns+`) VALUES (?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.Role, u.PasswordHash, formatTime(u.CreatedAt))
	var sqlErr *sqlite.Error
	if errors.As(err, &sqlErr) && sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return model.ErrEmailTaken
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// UserByEmail returns the user with the given (normalized) email, or model.ErrUserNotFound.
func (r *UserRepository) UserByEmail(ctx context.Context, email string) (model.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE email = ?`, email)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrUserNotFound
	}
	return u, err
}

// CreateSession stores a session for userID under the hash of its token.
// Logging in counts as being seen.
func (r *UserRepository) CreateSession(ctx context.Context, tokenHash, userID string, createdAt, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at, created_at, last_seen_at) VALUES (?, ?, ?, ?, ?)`,
		tokenHash, userID, formatTime(expiresAt), formatTime(createdAt), formatTime(createdAt))
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// UserBySession returns the owner of an unexpired session and when the session
// was last seen (zero if never recorded), or model.ErrUnauthenticated.
func (r *UserRepository) UserBySession(ctx context.Context, tokenHash string, now time.Time) (model.User, time.Time, error) {
	// RFC3339 UTC timestamps of equal precision sort correctly as strings.
	row := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.role, u.password_hash, u.created_at, s.last_seen_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`,
		tokenHash, formatTime(now))

	var (
		u           model.User
		created     string
		lastSeenRaw sql.NullString
		lastSeen    time.Time
	)
	err := row.Scan(&u.ID, &u.Email, &u.Role, &u.PasswordHash, &created, &lastSeenRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, time.Time{}, model.ErrUnauthenticated
	}
	if err != nil {
		return model.User{}, time.Time{}, fmt.Errorf("scan session user: %w", err)
	}
	if u.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return model.User{}, time.Time{}, fmt.Errorf("parse created_at of user %s: %w", u.ID, err)
	}
	if lastSeenRaw.Valid {
		if lastSeen, err = time.Parse(time.RFC3339, lastSeenRaw.String); err != nil {
			return model.User{}, time.Time{}, fmt.Errorf("parse last_seen_at of session: %w", err)
		}
	}
	return u, lastSeen, nil
}

// TouchSession records that a session was used at seenAt.
func (r *UserRepository) TouchSession(ctx context.Context, tokenHash string, seenAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`,
		formatTime(seenAt), tokenHash)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// UserStats counts users by role (every role), and for the given role only:
// its users, and those of them with an unexpired session (all, and those seen
// at or after onlineSince). ByRole only contains roles that have users.
func (r *UserRepository) UserStats(ctx context.Context, role model.Role, now, onlineSince time.Time) (model.UserStats, error) {
	stats := model.UserStats{ByRole: map[model.Role]int{}}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users WHERE role = ?1),
			(SELECT COUNT(DISTINCT s.user_id) FROM sessions s JOIN users u ON u.id = s.user_id
				WHERE u.role = ?1 AND s.expires_at > ?2),
			(SELECT COUNT(DISTINCT s.user_id) FROM sessions s JOIN users u ON u.id = s.user_id
				WHERE u.role = ?1 AND s.expires_at > ?2 AND s.last_seen_at >= ?3)`,
		role, formatTime(now), formatTime(onlineSince),
	).Scan(&stats.Total, &stats.WithActiveSession, &stats.Online)
	if err != nil {
		return model.UserStats{}, fmt.Errorf("count users and sessions: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `SELECT role, COUNT(*) FROM users GROUP BY role`)
	if err != nil {
		return model.UserStats{}, fmt.Errorf("count users by role: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			role model.Role
			n    int
		)
		if err := rows.Scan(&role, &n); err != nil {
			return model.UserStats{}, fmt.Errorf("scan role count: %w", err)
		}
		stats.ByRole[role] = n
	}
	if err := rows.Err(); err != nil {
		return model.UserStats{}, fmt.Errorf("iterate role counts: %w", err)
	}

	if stats.ActiveUsers, err = r.activeUsers(ctx, role, now, onlineSince); err != nil {
		return model.UserStats{}, err
	}
	return stats, nil
}

// activeUsers lists users of the given role with an unexpired session, most
// recently seen first (never-seen sessions last), capped at MaxActiveUsersListed.
func (r *UserRepository) activeUsers(ctx context.Context, role model.Role, now, onlineSince time.Time) ([]model.ActiveUser, error) {
	// In SQLite NULL sorts lowest, so DESC puts users never seen last.
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.email, MAX(s.last_seen_at) AS last_seen
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE u.role = ? AND s.expires_at > ?
		GROUP BY u.id, u.email
		ORDER BY last_seen DESC, u.email
		LIMIT ?`,
		role, formatTime(now), model.MaxActiveUsersListed)
	if err != nil {
		return nil, fmt.Errorf("query active users: %w", err)
	}
	defer rows.Close()

	since := formatTime(onlineSince)
	users := make([]model.ActiveUser, 0)
	for rows.Next() {
		var (
			u        model.ActiveUser
			lastSeen sql.NullString
		)
		if err := rows.Scan(&u.ID, &u.Email, &lastSeen); err != nil {
			return nil, fmt.Errorf("scan active user: %w", err)
		}
		if lastSeen.Valid {
			t, err := time.Parse(time.RFC3339, lastSeen.String)
			if err != nil {
				return nil, fmt.Errorf("parse last_seen_at of user %s: %w", u.ID, err)
			}
			u.LastSeenAt = &t
			// Same rule as the online count: seen at or after onlineSince.
			u.Online = lastSeen.String >= since
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active users: %w", err)
	}
	return users, nil
}

// DeleteSession removes a session. Deleting an unknown session is not an error.
func (r *UserRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes every session that expired at or before now.
func (r *UserRepository) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, formatTime(now)); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

// scanUser returns sql.ErrNoRows unchanged so callers can choose the error to report.
func scanUser(s scanner) (model.User, error) {
	var (
		u       model.User
		created string
	)
	err := s.Scan(&u.ID, &u.Email, &u.Role, &u.PasswordHash, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, err
	}
	if err != nil {
		return model.User{}, fmt.Errorf("scan user: %w", err)
	}
	if u.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return model.User{}, fmt.Errorf("parse created_at of user %s: %w", u.ID, err)
	}
	return u, nil
}

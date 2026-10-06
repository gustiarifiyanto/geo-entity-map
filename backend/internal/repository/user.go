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
func (r *UserRepository) CreateSession(ctx context.Context, tokenHash, userID string, createdAt, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		tokenHash, userID, formatTime(expiresAt), formatTime(createdAt))
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// UserBySession returns the owner of an unexpired session, or model.ErrUnauthenticated.
func (r *UserRepository) UserBySession(ctx context.Context, tokenHash string, now time.Time) (model.User, error) {
	// RFC3339 UTC timestamps of equal precision sort correctly as strings.
	row := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.role, u.password_hash, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`,
		tokenHash, formatTime(now))
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrUnauthenticated
	}
	return u, err
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

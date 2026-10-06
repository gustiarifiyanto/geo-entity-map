package database

import (
	"context"
	"database/sql"
	"fmt"
)

// type and status are intentionally not constrained in SQL: the allowed values
// live only in package model, so adding a type requires no migration.
const schema = `
CREATE TABLE IF NOT EXISTS entities (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	type        TEXT NOT NULL,
	status      TEXT NOT NULL,
	latitude    REAL NOT NULL CHECK (latitude BETWEEN -90 AND 90),
	longitude   REAL NOT NULL CHECK (longitude BETWEEN -180 AND 180),
	description TEXT NOT NULL DEFAULT '',
	attributes  TEXT,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities (type);
CREATE INDEX IF NOT EXISTS idx_entities_status ON entities (status);

-- Unlike entity types, roles are checked in SQL: a wrong role is a security
-- problem, and roles must match model.Roles (see TestSchemaAcceptsEveryRole).
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role          TEXT NOT NULL CHECK (role IN ('user', 'admin')),
	created_at    TEXT NOT NULL
);

-- token_hash is the SHA-256 of the cookie token, so a leaked database file
-- cannot be used to log in.
CREATE TABLE IF NOT EXISTS sessions (
	token_hash TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL,
	-- NULL for sessions created before this column existed.
	last_seen_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);
`

// Migrate creates the schema if it does not exist and adds columns that
// older databases lack. It is safe to run on every start.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	if err := addSessionsLastSeenAt(ctx, db); err != nil {
		return err
	}
	return nil
}

// addSessionsLastSeenAt adds sessions.last_seen_at to databases created before
// the dashboard, because CREATE TABLE IF NOT EXISTS leaves existing tables alone.
func addSessionsLastSeenAt(ctx context.Context, db *sql.DB) error {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'last_seen_at'`).Scan(&n)
	if err != nil {
		return fmt.Errorf("inspect sessions columns: %w", err)
	}
	if n > 0 {
		return nil
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN last_seen_at TEXT`); err != nil {
		return fmt.Errorf("add sessions.last_seen_at: %w", err)
	}
	return nil
}

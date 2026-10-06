package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "nested", "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

func countEntities(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM entities`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestSeedOnlyWhenEmpty(t *testing.T) {
	db := openTestDB(t)
	for i := range 2 {
		if err := Seed(context.Background(), db); err != nil {
			t.Fatalf("Seed run %d: %v", i+1, err)
		}
		if got, want := countEntities(t, db), len(seedEntities); got != want {
			t.Fatalf("after seed run %d: count = %d, want %d", i+1, got, want)
		}
	}
}

func TestSeedDataIsValid(t *testing.T) {
	db := openTestDB(t)
	if err := Seed(context.Background(), db); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	rows, err := db.Query(`SELECT name, type, status, attributes FROM entities`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			name, typ, status string
			attrs             sql.NullString
		)
		if err := rows.Scan(&name, &typ, &status, &attrs); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if !model.EntityType(typ).Valid() {
			t.Errorf("%s: invalid type %q", name, typ)
		}
		if !model.EntityStatus(status).Valid() {
			t.Errorf("%s: invalid status %q", name, status)
		}
		if attrs.Valid {
			var obj map[string]any
			if err := json.Unmarshal([]byte(attrs.String), &obj); err != nil {
				t.Errorf("%s: attributes is not a JSON object: %v", name, err)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
}

func insertUser(db *sql.DB, id, email, role string) error {
	_, err := db.Exec(`INSERT INTO users (id, email, password_hash, role, created_at)
		VALUES (?, ?, 'hash', ?, 'now')`, id, email, role)
	return err
}

func TestSchemaAcceptsEveryRole(t *testing.T) {
	db := openTestDB(t)
	for _, r := range model.Roles {
		if err := insertUser(db, "id-"+string(r), string(r)+"@example.com", string(r)); err != nil {
			t.Errorf("insert user with role %q: %v", r, err)
		}
	}
	if err := insertUser(db, "id-root", "root@example.com", "root"); err == nil {
		t.Error("insert user with role \"root\" succeeded, want CHECK constraint error")
	}
}

func TestSchemaRejectsDuplicateEmail(t *testing.T) {
	db := openTestDB(t)
	if err := insertUser(db, "a", "budi@example.com", "user"); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := insertUser(db, "b", "budi@example.com", "user"); err == nil {
		t.Fatal("second insert with the same email succeeded, want UNIQUE constraint error")
	}
}

func TestDeletingUserDeletesSessions(t *testing.T) {
	db := openTestDB(t)
	if err := insertUser(db, "u1", "budi@example.com", "user"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
		VALUES ('t1', 'u1', 'later', 'now')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = 'u1'`); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if n != 0 {
		t.Fatalf("sessions after deleting user = %d, want 0", n)
	}
}

// The users and sessions tables as they were before sessions.last_seen_at.
const schemaBeforeLastSeen = `
CREATE TABLE users (
	id            TEXT PRIMARY KEY,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role          TEXT NOT NULL CHECK (role IN ('user', 'admin')),
	created_at    TEXT NOT NULL
);
CREATE TABLE sessions (
	token_hash TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL
);
INSERT INTO users VALUES ('u1', 'budi@example.com', 'hash', 'user', 'now');
INSERT INTO sessions VALUES ('t1', 'u1', 'later', 'now');
`

func TestMigrateAddsLastSeenToExistingSessions(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "old.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(schemaBeforeLastSeen); err != nil {
		t.Fatalf("create old schema: %v", err)
	}

	// Twice: the second run must find the column and do nothing.
	for i := range 2 {
		if err := Migrate(ctx, db); err != nil {
			t.Fatalf("Migrate run %d: %v", i+1, err)
		}
	}

	var (
		userID   string
		lastSeen sql.NullString
	)
	err = db.QueryRow(`SELECT user_id, last_seen_at FROM sessions WHERE token_hash = 't1'`).Scan(&userID, &lastSeen)
	if err != nil {
		t.Fatalf("existing session after migration: %v", err)
	}
	if userID != "u1" || lastSeen.Valid {
		t.Errorf("session = (%q, %v), want the old row with last_seen_at NULL", userID, lastSeen)
	}
}

func TestSchemaRejectsOutOfRangeCoordinates(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO entities
		(id, name, type, status, latitude, longitude, created_at, updated_at)
		VALUES ('x', 'n', 'vehicle', 'active', 90.0001, 0, 'now', 'now')`)
	if err == nil {
		t.Fatal("insert with latitude 90.0001 succeeded, want CHECK constraint error")
	}
}

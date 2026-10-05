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

func TestSchemaRejectsOutOfRangeCoordinates(t *testing.T) {
	db := openTestDB(t)
	_, err := db.Exec(`INSERT INTO entities
		(id, name, type, status, latitude, longitude, created_at, updated_at)
		VALUES ('x', 'n', 'vehicle', 'active', 90.0001, 0, 'now', 'now')`)
	if err == nil {
		t.Fatal("insert with latitude 90.0001 succeeded, want CHECK constraint error")
	}
}

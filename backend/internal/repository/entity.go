// Package repository contains the SQL queries for entities.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

const entityColumns = `id, name, type, status, latitude, longitude, description, attributes, created_at, updated_at`

// EntityRepository reads and writes entities in SQLite.
type EntityRepository struct {
	db *sql.DB
}

// NewEntityRepository returns a repository backed by db.
func NewEntityRepository(db *sql.DB) *EntityRepository {
	return &EntityRepository{db: db}
}

// List returns entities matching f, newest first. It never returns a nil slice.
func (r *EntityRepository) List(ctx context.Context, f model.EntityFilter) ([]model.Entity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+entityColumns+` FROM entities
		WHERE (?1 = '' OR type = ?1) AND (?2 = '' OR status = ?2)
		ORDER BY created_at DESC, name`,
		f.Type, f.Status)
	if err != nil {
		return nil, fmt.Errorf("query entities: %w", err)
	}
	defer rows.Close()

	entities := make([]model.Entity, 0)
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entities: %w", err)
	}
	return entities, nil
}

// Get returns the entity with the given id, or model.ErrNotFound.
func (r *EntityRepository) Get(ctx context.Context, id string) (model.Entity, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+entityColumns+` FROM entities WHERE id = ?`, id)
	return scanEntity(row)
}

// Create inserts e.
func (r *EntityRepository) Create(ctx context.Context, e model.Entity) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO entities (`+entityColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Name, e.Type, e.Status, e.Latitude, e.Longitude, e.Description,
		nullableJSON(e.Attributes), formatTime(e.CreatedAt), formatTime(e.UpdatedAt))
	if err != nil {
		return fmt.Errorf("insert entity: %w", err)
	}
	return nil
}

// Update overwrites every editable field of the entity with e.ID and returns
// the stored entity, or model.ErrNotFound. CreatedAt is left unchanged.
func (r *EntityRepository) Update(ctx context.Context, e model.Entity) (model.Entity, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE entities
		SET name = ?, type = ?, status = ?, latitude = ?, longitude = ?,
			description = ?, attributes = ?, updated_at = ?
		WHERE id = ?
		RETURNING `+entityColumns,
		e.Name, e.Type, e.Status, e.Latitude, e.Longitude,
		e.Description, nullableJSON(e.Attributes), formatTime(e.UpdatedAt), e.ID)
	return scanEntity(row)
}

// UpdateLocation changes only the coordinates and returns the stored entity,
// or model.ErrNotFound.
func (r *EntityRepository) UpdateLocation(ctx context.Context, id string, lat, lng float64, updatedAt time.Time) (model.Entity, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE entities SET latitude = ?, longitude = ?, updated_at = ?
		WHERE id = ?
		RETURNING `+entityColumns,
		lat, lng, formatTime(updatedAt), id)
	return scanEntity(row)
}

// Delete removes the entity with the given id, or returns model.ErrNotFound.
func (r *EntityRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM entities WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete entity: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete entity rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEntity(s scanner) (model.Entity, error) {
	var (
		e                model.Entity
		attrs            sql.NullString
		created, updated string
	)
	err := s.Scan(&e.ID, &e.Name, &e.Type, &e.Status, &e.Latitude, &e.Longitude,
		&e.Description, &attrs, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Entity{}, model.ErrNotFound
	}
	if err != nil {
		return model.Entity{}, fmt.Errorf("scan entity: %w", err)
	}

	if attrs.Valid {
		e.Attributes = json.RawMessage(attrs.String)
	}
	if e.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return model.Entity{}, fmt.Errorf("parse created_at of %s: %w", e.ID, err)
	}
	if e.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
		return model.Entity{}, fmt.Errorf("parse updated_at of %s: %w", e.ID, err)
	}
	return e, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// nullableJSON stores missing attributes as SQL NULL.
func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

// GeofenceRecord is a stored zone plus its entity's type and current position,
// which decide whether the entity is inside.
type GeofenceRecord struct {
	EntityID        string
	EntityType      model.EntityType
	EntityLatitude  float64
	EntityLongitude float64
	CenterLatitude  float64
	CenterLongitude float64
	RadiusM         float64
	UpdatedAt       time.Time
}

// GeofenceRepository reads and writes operating zones in SQLite.
type GeofenceRepository struct {
	db *sql.DB
}

// NewGeofenceRepository returns a repository backed by db.
func NewGeofenceRepository(db *sql.DB) *GeofenceRepository {
	return &GeofenceRepository{db: db}
}

const geofenceSelect = `
	SELECT g.entity_id, e.type, e.latitude, e.longitude, g.center_latitude, g.center_longitude, g.radius_m, g.updated_at
	FROM geofences g JOIN entities e ON e.id = g.entity_id`

// Get returns the zone of one entity, or found=false when it has none.
func (r *GeofenceRepository) Get(ctx context.Context, entityID string) (GeofenceRecord, bool, error) {
	rec, err := scanGeofence(r.db.QueryRowContext(ctx, geofenceSelect+` WHERE g.entity_id = ?`, entityID))
	if errors.Is(err, sql.ErrNoRows) {
		return GeofenceRecord{}, false, nil
	}
	return rec, err == nil, err
}

// List returns every stored zone.
func (r *GeofenceRepository) List(ctx context.Context) ([]GeofenceRecord, error) {
	rows, err := r.db.QueryContext(ctx, geofenceSelect)
	if err != nil {
		return nil, fmt.Errorf("query geofences: %w", err)
	}
	defer rows.Close()

	records := make([]GeofenceRecord, 0)
	for rows.Next() {
		rec, err := scanGeofence(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate geofences: %w", err)
	}
	return records, nil
}

// Put creates or replaces the zone of an entity.
func (r *GeofenceRepository) Put(ctx context.Context, entityID string, in model.GeofenceInput, updatedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO geofences (entity_id, center_latitude, center_longitude, radius_m, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (entity_id) DO UPDATE SET
			center_latitude = excluded.center_latitude,
			center_longitude = excluded.center_longitude,
			radius_m = excluded.radius_m,
			updated_at = excluded.updated_at`,
		entityID, *in.CenterLatitude, *in.CenterLongitude, *in.RadiusM, formatTime(updatedAt))
	if err != nil {
		return fmt.Errorf("upsert geofence: %w", err)
	}
	return nil
}

// Delete removes the zone of an entity. Deleting a missing zone is not an error.
func (r *GeofenceRepository) Delete(ctx context.Context, entityID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM geofences WHERE entity_id = ?`, entityID); err != nil {
		return fmt.Errorf("delete geofence: %w", err)
	}
	return nil
}

// scanGeofence returns sql.ErrNoRows unchanged.
func scanGeofence(s scanner) (GeofenceRecord, error) {
	var (
		rec     GeofenceRecord
		updated string
	)
	err := s.Scan(&rec.EntityID, &rec.EntityType, &rec.EntityLatitude, &rec.EntityLongitude,
		&rec.CenterLatitude, &rec.CenterLongitude, &rec.RadiusM, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return GeofenceRecord{}, err
	}
	if err != nil {
		return GeofenceRecord{}, fmt.Errorf("scan geofence: %w", err)
	}
	if rec.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
		return GeofenceRecord{}, fmt.Errorf("parse updated_at of geofence %s: %w", rec.EntityID, err)
	}
	return rec, nil
}

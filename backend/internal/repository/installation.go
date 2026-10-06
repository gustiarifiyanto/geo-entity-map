package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

// InstallationRecord is a stored installation schedule plus the type of its
// entity, so callers can skip entities whose type changed since.
type InstallationRecord struct {
	EntityID    string
	EntityType  model.EntityType
	StartedOn   string
	TargetOn    string
	CompletedOn *string
	UpdatedAt   time.Time
}

// InstallationRepository reads and writes installation schedules in SQLite.
type InstallationRepository struct {
	db *sql.DB
}

// NewInstallationRepository returns a repository backed by db.
func NewInstallationRepository(db *sql.DB) *InstallationRepository {
	return &InstallationRepository{db: db}
}

const installationSelect = `
	SELECT i.entity_id, e.type, i.started_on, i.target_on, i.completed_on, i.updated_at
	FROM facility_installations i JOIN entities e ON e.id = i.entity_id`

// Get returns the schedule of one entity, or found=false when it has none.
func (r *InstallationRepository) Get(ctx context.Context, entityID string) (InstallationRecord, bool, error) {
	rec, err := scanInstallation(r.db.QueryRowContext(ctx, installationSelect+` WHERE i.entity_id = ?`, entityID))
	if errors.Is(err, sql.ErrNoRows) {
		return InstallationRecord{}, false, nil
	}
	return rec, err == nil, err
}

// List returns every stored schedule.
func (r *InstallationRepository) List(ctx context.Context) ([]InstallationRecord, error) {
	rows, err := r.db.QueryContext(ctx, installationSelect)
	if err != nil {
		return nil, fmt.Errorf("query installations: %w", err)
	}
	defer rows.Close()

	records := make([]InstallationRecord, 0)
	for rows.Next() {
		rec, err := scanInstallation(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate installations: %w", err)
	}
	return records, nil
}

// Put creates or replaces the schedule of an entity.
func (r *InstallationRepository) Put(ctx context.Context, in model.InstallationInput, entityID string, updatedAt time.Time) error {
	var completed sql.NullString
	if in.CompletedOn != nil {
		completed = sql.NullString{String: *in.CompletedOn, Valid: true}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO facility_installations (entity_id, started_on, target_on, completed_on, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (entity_id) DO UPDATE SET
			started_on = excluded.started_on,
			target_on = excluded.target_on,
			completed_on = excluded.completed_on,
			updated_at = excluded.updated_at`,
		entityID, in.StartedOn, in.TargetOn, completed, formatTime(updatedAt))
	if err != nil {
		return fmt.Errorf("upsert installation: %w", err)
	}
	return nil
}

// Delete removes the schedule of an entity. Deleting a missing schedule is not an error.
func (r *InstallationRepository) Delete(ctx context.Context, entityID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM facility_installations WHERE entity_id = ?`, entityID); err != nil {
		return fmt.Errorf("delete installation: %w", err)
	}
	return nil
}

// scanInstallation returns sql.ErrNoRows unchanged.
func scanInstallation(s scanner) (InstallationRecord, error) {
	var (
		rec       InstallationRecord
		completed sql.NullString
		updated   string
	)
	err := s.Scan(&rec.EntityID, &rec.EntityType, &rec.StartedOn, &rec.TargetOn, &completed, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return InstallationRecord{}, err
	}
	if err != nil {
		return InstallationRecord{}, fmt.Errorf("scan installation: %w", err)
	}
	if completed.Valid {
		rec.CompletedOn = &completed.String
	}
	if rec.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
		return InstallationRecord{}, fmt.Errorf("parse updated_at of installation %s: %w", rec.EntityID, err)
	}
	return rec, nil
}

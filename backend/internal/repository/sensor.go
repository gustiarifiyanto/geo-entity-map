package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

// SensorRecord is a stored sensor config plus its entity's type, so callers
// can skip entities whose type changed since.
type SensorRecord struct {
	model.SensorConfig
	EntityType model.EntityType
}

// SensorRepository reads and writes sensor configs and readings in SQLite.
type SensorRepository struct {
	db *sql.DB
}

// NewSensorRepository returns a repository backed by db.
func NewSensorRepository(db *sql.DB) *SensorRepository {
	return &SensorRepository{db: db}
}

const sensorSelect = `
	SELECT s.entity_id, e.type, s.metric, s.api_key_hash IS NOT NULL, s.key_created_at, s.updated_at
	FROM sensor_configs s JOIN entities e ON e.id = s.entity_id`

// Get returns the config of one entity, or found=false when it has none.
func (r *SensorRepository) Get(ctx context.Context, entityID string) (SensorRecord, bool, error) {
	rec, err := scanSensor(r.db.QueryRowContext(ctx, sensorSelect+` WHERE s.entity_id = ?`, entityID))
	if errors.Is(err, sql.ErrNoRows) {
		return SensorRecord{}, false, nil
	}
	return rec, err == nil, err
}

// List returns every sensor config.
func (r *SensorRepository) List(ctx context.Context) ([]SensorRecord, error) {
	rows, err := r.db.QueryContext(ctx, sensorSelect)
	if err != nil {
		return nil, fmt.Errorf("query sensors: %w", err)
	}
	defer rows.Close()

	records := make([]SensorRecord, 0)
	for rows.Next() {
		rec, err := scanSensor(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sensors: %w", err)
	}
	return records, nil
}

// PutMetric creates the config of an entity or changes its metric. An
// existing API key is kept.
func (r *SensorRepository) PutMetric(ctx context.Context, entityID string, metric model.Metric, updatedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sensor_configs (entity_id, metric, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (entity_id) DO UPDATE SET metric = excluded.metric, updated_at = excluded.updated_at`,
		entityID, metric, formatTime(updatedAt))
	if err != nil {
		return fmt.Errorf("upsert sensor: %w", err)
	}
	return nil
}

// SetKey replaces the API key hash of an existing config, or returns
// found=false when the entity has no config yet.
func (r *SensorRepository) SetKey(ctx context.Context, entityID, keyHash string, createdAt time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE sensor_configs SET api_key_hash = ?, key_created_at = ?, updated_at = ? WHERE entity_id = ?`,
		keyHash, formatTime(createdAt), formatTime(createdAt), entityID)
	if err != nil {
		return false, fmt.Errorf("set sensor key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("set sensor key rows affected: %w", err)
	}
	return n > 0, nil
}

// Delete removes the config (and key) of an entity. Readings are kept until they expire.
func (r *SensorRepository) Delete(ctx context.Context, entityID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sensor_configs WHERE entity_id = ?`, entityID); err != nil {
		return fmt.Errorf("delete sensor: %w", err)
	}
	return nil
}

// EntityByKey returns the entity whose device key has the given hash, or
// found=false when no device has it.
func (r *SensorRepository) EntityByKey(ctx context.Context, keyHash string) (string, bool, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT entity_id FROM sensor_configs WHERE api_key_hash = ?`, keyHash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("find device by key: %w", err)
	}
	return id, true, nil
}

// AddReadings stores readings of one entity and metric in a single transaction.
func (r *SensorRepository) AddReadings(ctx context.Context, entityID string, metric model.Metric, readings []model.Reading, receivedAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin readings transaction: %w", err)
	}
	defer tx.Rollback()

	for _, rd := range readings {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO sensor_readings (entity_id, metric, value, recorded_at, received_at)
			VALUES (?, ?, ?, ?, ?)`,
			entityID, metric, rd.Value, formatTime(rd.RecordedAt), formatTime(receivedAt))
		if err != nil {
			return fmt.Errorf("insert reading: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit readings: %w", err)
	}
	return nil
}

// Readings returns the readings of an entity for a metric recorded at or
// after since, oldest first. It never returns a nil slice.
func (r *SensorRepository) Readings(ctx context.Context, entityID string, metric model.Metric, since time.Time) ([]model.Reading, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT value, recorded_at FROM sensor_readings
		WHERE entity_id = ? AND metric = ? AND recorded_at >= ?
		ORDER BY recorded_at, id`,
		entityID, metric, formatTime(since))
	if err != nil {
		return nil, fmt.Errorf("query readings: %w", err)
	}
	defer rows.Close()
	return scanReadings(rows)
}

// LatestReading returns the newest reading of an entity for a metric, or nil.
func (r *SensorRepository) LatestReading(ctx context.Context, entityID string, metric model.Metric) (*model.Reading, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT value, recorded_at FROM sensor_readings
		WHERE entity_id = ? AND metric = ?
		ORDER BY recorded_at DESC, id DESC LIMIT 1`,
		entityID, metric)
	if err != nil {
		return nil, fmt.Errorf("query latest reading: %w", err)
	}
	defer rows.Close()
	list, err := scanReadings(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

// DeleteReadingsBefore removes readings recorded before cutoff and reports how many.
func (r *SensorRepository) DeleteReadingsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sensor_readings WHERE recorded_at < ?`, formatTime(cutoff))
	if err != nil {
		return 0, fmt.Errorf("delete old readings: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete old readings rows affected: %w", err)
	}
	return n, nil
}

func scanReadings(rows *sql.Rows) ([]model.Reading, error) {
	readings := make([]model.Reading, 0)
	for rows.Next() {
		var (
			rd model.Reading
			at string
		)
		if err := rows.Scan(&rd.Value, &at); err != nil {
			return nil, fmt.Errorf("scan reading: %w", err)
		}
		t, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return nil, fmt.Errorf("parse recorded_at: %w", err)
		}
		rd.RecordedAt = t
		readings = append(readings, rd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate readings: %w", err)
	}
	return readings, nil
}

// scanSensor returns sql.ErrNoRows unchanged.
func scanSensor(s scanner) (SensorRecord, error) {
	var (
		rec        SensorRecord
		keyCreated sql.NullString
		updated    string
	)
	err := s.Scan(&rec.EntityID, &rec.EntityType, &rec.Metric, &rec.HasAPIKey, &keyCreated, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return SensorRecord{}, err
	}
	if err != nil {
		return SensorRecord{}, fmt.Errorf("scan sensor: %w", err)
	}
	if keyCreated.Valid {
		t, err := time.Parse(time.RFC3339, keyCreated.String)
		if err != nil {
			return SensorRecord{}, fmt.Errorf("parse key_created_at of %s: %w", rec.EntityID, err)
		}
		rec.KeyCreatedAt = &t
	}
	if rec.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
		return SensorRecord{}, fmt.Errorf("parse updated_at of sensor %s: %w", rec.EntityID, err)
	}
	return rec, nil
}

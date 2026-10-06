package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

const photoColumns = `id, entity_id, content_type, size_bytes, created_at`

// PhotoRepository reads and writes photo metadata in SQLite.
type PhotoRepository struct {
	db *sql.DB
}

// NewPhotoRepository returns a repository backed by db.
func NewPhotoRepository(db *sql.DB) *PhotoRepository {
	return &PhotoRepository{db: db}
}

// CreatePhoto inserts p unless its entity already has limit photos, in which
// case it returns model.ErrTooManyPhotos. The count and the insert are one
// statement, so concurrent uploads cannot exceed the limit.
func (r *PhotoRepository) CreatePhoto(ctx context.Context, p model.Photo, limit int) error {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO entity_photos (`+photoColumns+`)
		SELECT ?1, ?2, ?3, ?4, ?5
		WHERE (SELECT COUNT(*) FROM entity_photos WHERE entity_id = ?2) < ?6`,
		p.ID, p.EntityID, p.ContentType, p.SizeBytes, formatTime(p.CreatedAt), limit)
	if err != nil {
		return fmt.Errorf("insert photo: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("insert photo rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrTooManyPhotos
	}
	return nil
}

// ListPhotos returns an entity's photos, oldest first. It never returns a nil slice.
func (r *PhotoRepository) ListPhotos(ctx context.Context, entityID string) ([]model.Photo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+photoColumns+` FROM entity_photos
		WHERE entity_id = ?
		ORDER BY created_at, rowid`, entityID)
	if err != nil {
		return nil, fmt.Errorf("query photos: %w", err)
	}
	defer rows.Close()

	photos := make([]model.Photo, 0)
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate photos: %w", err)
	}
	return photos, nil
}

// GetPhoto returns one photo, or model.ErrPhotoNotFound.
func (r *PhotoRepository) GetPhoto(ctx context.Context, id string) (model.Photo, error) {
	return scanPhoto(r.db.QueryRowContext(ctx, `SELECT `+photoColumns+` FROM entity_photos WHERE id = ?`, id))
}

// DeletePhoto removes one photo and returns what was deleted, or model.ErrPhotoNotFound.
func (r *PhotoRepository) DeletePhoto(ctx context.Context, id string) (model.Photo, error) {
	return scanPhoto(r.db.QueryRowContext(ctx, `DELETE FROM entity_photos WHERE id = ? RETURNING `+photoColumns, id))
}

func scanPhoto(s scanner) (model.Photo, error) {
	var (
		p       model.Photo
		created string
	)
	err := s.Scan(&p.ID, &p.EntityID, &p.ContentType, &p.SizeBytes, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Photo{}, model.ErrPhotoNotFound
	}
	if err != nil {
		return model.Photo{}, fmt.Errorf("scan photo: %w", err)
	}
	if p.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return model.Photo{}, fmt.Errorf("parse created_at of photo %s: %w", p.ID, err)
	}
	return p, nil
}

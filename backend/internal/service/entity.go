// Package service contains the entity business logic.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

// Repository is the storage the service depends on.
type Repository interface {
	List(ctx context.Context, f model.EntityFilter) ([]model.Entity, error)
	Get(ctx context.Context, id string) (model.Entity, error)
	Create(ctx context.Context, e model.Entity) error
	Update(ctx context.Context, e model.Entity) (model.Entity, error)
	UpdateLocation(ctx context.Context, id string, lat, lng float64, updatedAt time.Time) (model.Entity, error)
	Delete(ctx context.Context, id string) error
}

// EntityService implements entity use cases. Inputs passed to it must
// already have been validated.
type EntityService struct {
	repo  Repository
	files EntityFiles
	now   func() time.Time
}

// EntityFiles removes the files stored for an entity (its photos).
type EntityFiles interface {
	RemoveEntity(entityID string) error
}

// NewEntityService returns a service backed by repo.
func NewEntityService(repo Repository, files EntityFiles) *EntityService {
	return &EntityService{
		repo:  repo,
		files: files,
		// Timestamps are stored with second precision (RFC3339).
		now: func() time.Time { return time.Now().UTC().Truncate(time.Second) },
	}
}

// List returns entities matching f.
func (s *EntityService) List(ctx context.Context, f model.EntityFilter) ([]model.Entity, error) {
	return s.repo.List(ctx, f)
}

// Get returns one entity, or model.ErrNotFound.
func (s *EntityService) Get(ctx context.Context, id string) (model.Entity, error) {
	if !validID(id) {
		return model.Entity{}, model.ErrNotFound
	}
	return s.repo.Get(ctx, id)
}

// Create stores a new entity built from a validated input.
func (s *EntityService) Create(ctx context.Context, in model.EntityInput) (model.Entity, error) {
	now := s.now()
	e := fromInput(uuid.NewString(), in)
	e.CreatedAt, e.UpdatedAt = now, now
	if err := s.repo.Create(ctx, e); err != nil {
		return model.Entity{}, err
	}
	return e, nil
}

// Update replaces every editable field of an entity, or returns model.ErrNotFound.
func (s *EntityService) Update(ctx context.Context, id string, in model.EntityInput) (model.Entity, error) {
	if !validID(id) {
		return model.Entity{}, model.ErrNotFound
	}
	e := fromInput(id, in)
	e.UpdatedAt = s.now()
	return s.repo.Update(ctx, e)
}

// UpdateLocation moves an entity, or returns model.ErrNotFound.
func (s *EntityService) UpdateLocation(ctx context.Context, id string, in model.LocationInput) (model.Entity, error) {
	if !validID(id) {
		return model.Entity{}, model.ErrNotFound
	}
	return s.repo.UpdateLocation(ctx, id, *in.Latitude, *in.Longitude, s.now())
}

// Delete removes an entity, or returns model.ErrNotFound.
func (s *EntityService) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return model.ErrNotFound
	}
	// The photo rows go with the entity (ON DELETE CASCADE); the files are
	// removed here.
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.files.RemoveEntity(id)
}

func fromInput(id string, in model.EntityInput) model.Entity {
	return model.Entity{
		ID:          id,
		Name:        in.Name,
		Type:        in.Type,
		Status:      in.Status,
		Latitude:    *in.Latitude,
		Longitude:   *in.Longitude,
		Description: in.Description,
		Attributes:  in.Attributes,
	}
}

// validID reports whether id is a UUID, so malformed IDs become 404 without a query.
func validID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

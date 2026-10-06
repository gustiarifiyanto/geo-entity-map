package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
)

// GeofenceRepository is the zone storage the geofence service depends on.
type GeofenceRepository interface {
	Get(ctx context.Context, entityID string) (repository.GeofenceRecord, bool, error)
	List(ctx context.Context) ([]repository.GeofenceRecord, error)
	Put(ctx context.Context, entityID string, in model.GeofenceInput, updatedAt time.Time) error
	Delete(ctx context.Context, entityID string) error
}

// GeofenceService manages operating zones and judges, for each entity's
// current position, whether it is inside.
type GeofenceService struct {
	repo     GeofenceRepository
	entities EntityLookup
	now      func() time.Time
}

// NewGeofenceService returns a geofence service.
func NewGeofenceService(repo GeofenceRepository, entities EntityLookup) *GeofenceService {
	return &GeofenceService{
		repo:     repo,
		entities: entities,
		now:      func() time.Time { return time.Now().UTC().Truncate(time.Second) },
	}
}

// Get returns an entity's zone, nil when none, model.ErrNotFound for an
// unknown entity, or model.ErrNoGeofence if its type has no zone.
func (s *GeofenceService) Get(ctx context.Context, entityID string) (*model.Geofence, error) {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return nil, err
	}
	rec, found, err := s.repo.Get(ctx, entityID)
	if err != nil || !found {
		return nil, err
	}
	g := evaluate(rec)
	return &g, nil
}

// Put sets an entity's zone from a validated input and returns it.
func (s *GeofenceService) Put(ctx context.Context, entityID string, in model.GeofenceInput) (model.Geofence, error) {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return model.Geofence{}, err
	}
	if err := s.repo.Put(ctx, entityID, in, s.now()); err != nil {
		return model.Geofence{}, err
	}
	g, err := s.Get(ctx, entityID)
	if err != nil {
		return model.Geofence{}, err
	}
	if g == nil {
		return model.Geofence{}, fmt.Errorf("geofence of %s missing right after saving it", entityID)
	}
	return *g, nil
}

// Delete removes an entity's zone. Removing a missing zone is not an error.
func (s *GeofenceService) Delete(ctx context.Context, entityID string) error {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, entityID)
}

// List returns the zones of every entity whose type has the geofence
// capability: entities outside their zone first (farthest past the edge
// first), then the rest by distance from the center.
func (s *GeofenceService) List(ctx context.Context) ([]model.Geofence, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]model.Geofence, 0, len(records))
	for _, rec := range records {
		// A zone outlives a type change; it only counts while the type supports it.
		if rec.EntityType.Has(model.CapGeofence) {
			list = append(list, evaluate(rec))
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Inside != b.Inside {
			return !a.Inside
		}
		return a.DistanceM-a.RadiusM > b.DistanceM-b.RadiusM
	})
	return list, nil
}

func (s *GeofenceService) checkEntity(ctx context.Context, entityID string) error {
	e, err := s.entities.Get(ctx, entityID)
	if err != nil {
		return err
	}
	if !e.Type.Has(model.CapGeofence) {
		return model.ErrNoGeofence
	}
	return nil
}

// evaluate judges the entity's current position against its zone.
func evaluate(rec repository.GeofenceRecord) model.Geofence {
	distance, inside := model.ZoneStatus(rec.CenterLatitude, rec.CenterLongitude, rec.RadiusM,
		rec.EntityLatitude, rec.EntityLongitude)
	return model.Geofence{
		EntityID:        rec.EntityID,
		CenterLatitude:  rec.CenterLatitude,
		CenterLongitude: rec.CenterLongitude,
		RadiusM:         rec.RadiusM,
		DistanceM:       distance,
		Inside:          inside,
		UpdatedAt:       rec.UpdatedAt,
	}
}

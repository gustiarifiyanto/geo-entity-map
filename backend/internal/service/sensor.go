package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
)

// deviceKeyPrefix makes device keys recognizable (e.g. in a leaked config file).
const deviceKeyPrefix = "gem_"

// SensorRepository is the sensor storage the sensor service depends on.
type SensorRepository interface {
	Get(ctx context.Context, entityID string) (repository.SensorRecord, bool, error)
	List(ctx context.Context) ([]repository.SensorRecord, error)
	PutMetric(ctx context.Context, entityID string, metric model.Metric, updatedAt time.Time) error
	SetKey(ctx context.Context, entityID, keyHash string, createdAt time.Time) (bool, error)
	Delete(ctx context.Context, entityID string) error
	EntityByKey(ctx context.Context, keyHash string) (string, bool, error)
	AddReadings(ctx context.Context, entityID string, metric model.Metric, readings []model.Reading, receivedAt time.Time) error
	Readings(ctx context.Context, entityID string, metric model.Metric, since time.Time) ([]model.Reading, error)
	LatestReading(ctx context.Context, entityID string, metric model.Metric) (*model.Reading, error)
	DeleteReadingsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// SensorService manages IoT sensors: their metric, device keys and readings.
// Readings must be validated by the caller (validation.Reading) before Record.
type SensorService struct {
	repo     SensorRepository
	entities EntityLookup
	now      func() time.Time
}

// NewSensorService returns a sensor service.
func NewSensorService(repo SensorRepository, entities EntityLookup) *SensorService {
	return &SensorService{
		repo:     repo,
		entities: entities,
		now:      func() time.Time { return time.Now().UTC().Truncate(time.Second) },
	}
}

// Now is the service's current time, used to validate reading timestamps.
func (s *SensorService) Now() time.Time {
	return s.now()
}

// Get returns an entity's sensor config, nil when none, model.ErrNotFound for
// an unknown entity, or model.ErrNoReadings if its type has no sensor.
func (s *SensorService) Get(ctx context.Context, entityID string) (*model.SensorConfig, error) {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return nil, err
	}
	rec, found, err := s.repo.Get(ctx, entityID)
	if err != nil || !found {
		return nil, err
	}
	return &rec.SensorConfig, nil
}

// PutMetric sets the metric of an entity's sensor from a validated input.
func (s *SensorService) PutMetric(ctx context.Context, entityID string, in model.SensorInput) (model.SensorConfig, error) {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return model.SensorConfig{}, err
	}
	if err := s.repo.PutMetric(ctx, entityID, in.Metric, s.now()); err != nil {
		return model.SensorConfig{}, err
	}
	cfg, err := s.Get(ctx, entityID)
	if err != nil {
		return model.SensorConfig{}, err
	}
	if cfg == nil {
		return model.SensorConfig{}, fmt.Errorf("sensor of %s missing right after saving it", entityID)
	}
	return *cfg, nil
}

// Delete removes an entity's sensor config and key. Readings expire on their own.
func (s *SensorService) Delete(ctx context.Context, entityID string) error {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, entityID)
}

// CreateKey makes a new device API key for an entity, replacing any old one.
// The raw key is returned only here; the database keeps its hash.
func (s *SensorService) CreateKey(ctx context.Context, entityID string) (model.DeviceKey, error) {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return model.DeviceKey{}, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return model.DeviceKey{}, fmt.Errorf("generate device key: %w", err)
	}
	key := deviceKeyPrefix + base64.RawURLEncoding.EncodeToString(raw)

	now := s.now()
	found, err := s.repo.SetKey(ctx, entityID, hashToken(key), now)
	if err != nil {
		return model.DeviceKey{}, err
	}
	if !found {
		return model.DeviceKey{}, model.ErrNoSensorMetric
	}
	return model.DeviceKey{APIKey: key, KeyCreatedAt: now}, nil
}

// DeviceMetric checks a device's key against the device in the URL and
// returns the metric it reports. Any mismatch is model.ErrInvalidDeviceKey,
// without saying what was wrong.
func (s *SensorService) DeviceMetric(ctx context.Context, entityID, key string) (model.MetricSpec, error) {
	if key == "" {
		return model.MetricSpec{}, model.ErrInvalidDeviceKey
	}
	owner, found, err := s.repo.EntityByKey(ctx, hashToken(key))
	if err != nil {
		return model.MetricSpec{}, err
	}
	if !found || owner != entityID {
		return model.MetricSpec{}, model.ErrInvalidDeviceKey
	}
	rec, found, err := s.repo.Get(ctx, entityID)
	if err != nil {
		return model.MetricSpec{}, err
	}
	spec, known := rec.Metric.Spec()
	// The entity's type may have changed since the key was made.
	if !found || !known || !rec.EntityType.Has(model.CapReadings) {
		return model.MetricSpec{}, model.ErrInvalidDeviceKey
	}
	return spec, nil
}

// Record stores validated readings of an entity's current metric.
func (s *SensorService) Record(ctx context.Context, entityID string, metric model.Metric, readings []model.Reading) error {
	return s.repo.AddReadings(ctx, entityID, metric, readings, s.now())
}

// Readings returns the readings of the last `hours` hours for the entity's
// current metric, or nil when it has no sensor config.
func (s *SensorService) Readings(ctx context.Context, entityID string, hours int) (*model.Readings, error) {
	cfg, err := s.Get(ctx, entityID)
	if err != nil || cfg == nil {
		return nil, err
	}
	spec, ok := cfg.Metric.Spec()
	if !ok {
		return nil, fmt.Errorf("sensor of %s has unknown metric %q", entityID, cfg.Metric)
	}
	list, err := s.repo.Readings(ctx, entityID, cfg.Metric, s.now().Add(-time.Duration(hours)*time.Hour))
	if err != nil {
		return nil, err
	}
	latest, err := s.repo.LatestReading(ctx, entityID, cfg.Metric)
	if err != nil {
		return nil, err
	}
	return &model.Readings{Metric: spec, Readings: list, Latest: latest}, nil
}

// Sensors returns every configured sensor whose entity type still has the
// readings capability, with its metric. Used by the simulator.
func (s *SensorService) Sensors(ctx context.Context) ([]model.SensorConfig, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	configs := make([]model.SensorConfig, 0, len(records))
	for _, rec := range records {
		if rec.EntityType.Has(model.CapReadings) {
			configs = append(configs, rec.SensorConfig)
		}
	}
	return configs, nil
}

// LatestReading returns the newest reading of an entity's metric, or nil.
func (s *SensorService) LatestReading(ctx context.Context, entityID string, metric model.Metric) (*model.Reading, error) {
	return s.repo.LatestReading(ctx, entityID, metric)
}

// Prune deletes readings older than model.ReadingRetention.
func (s *SensorService) Prune(ctx context.Context) (int64, error) {
	return s.repo.DeleteReadingsBefore(ctx, s.now().Add(-model.ReadingRetention))
}

func (s *SensorService) checkEntity(ctx context.Context, entityID string) error {
	e, err := s.entities.Get(ctx, entityID)
	if err != nil {
		return err
	}
	if !e.Type.Has(model.CapReadings) {
		return model.ErrNoReadings
	}
	return nil
}

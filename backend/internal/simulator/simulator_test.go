package simulator

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/database"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/storage"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

type fixture struct {
	db       *sql.DB
	sim      *Simulator
	entities *service.EntityService
	sensors  *service.SensorService
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	files, err := storage.NewFiles(filepath.Join(t.TempDir(), "uploads"))
	if err != nil {
		t.Fatalf("files: %v", err)
	}
	val, err := validation.New()
	if err != nil {
		t.Fatalf("validator: %v", err)
	}
	entities := service.NewEntityService(repository.NewEntityRepository(db), files)
	sensors := service.NewSensorService(repository.NewSensorRepository(db), entities)
	return fixture{db: db, sim: New(sensors, val, 42), entities: entities, sensors: sensors}
}

func (f fixture) device(t *testing.T, typ model.EntityType, metric model.Metric) string {
	t.Helper()
	ctx := context.Background()
	lat, lng := -6.1, 106.8
	e, err := f.entities.Create(ctx, model.EntityInput{Name: "Device", Type: typ, Status: model.StatusActive, Latitude: &lat, Longitude: &lng})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	if typ.Has(model.CapReadings) {
		if _, err := f.sensors.PutMetric(ctx, e.ID, model.SensorInput{Metric: metric}); err != nil {
			t.Fatalf("put metric: %v", err)
		}
	}
	return e.ID
}

func (f fixture) count(t *testing.T, entityID string) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM sensor_readings WHERE entity_id = ?`, entityID).Scan(&n); err != nil {
		t.Fatalf("count readings: %v", err)
	}
	return n
}

func TestTickBackfillsThenAddsOneReading(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ids := map[model.Metric]string{}
	for _, m := range model.Metrics {
		ids[m.ID] = f.device(t, model.TypeIoTDevice, m.ID)
	}

	if err := f.sim.Tick(ctx); err != nil {
		t.Fatalf("first tick: %v", err)
	}
	wantBackfill := int(backfillWindow / backfillStep)
	for metric, id := range ids {
		if got := f.count(t, id); got != wantBackfill {
			t.Errorf("%s after first tick: %d readings, want %d (a day of history)", metric, got, wantBackfill)
		}
	}

	if err := f.sim.Tick(ctx); err != nil {
		t.Fatalf("second tick: %v", err)
	}
	for metric, id := range ids {
		if got := f.count(t, id); got != wantBackfill+1 {
			t.Errorf("%s after second tick: %d readings, want %d", metric, got, wantBackfill+1)
		}
	}

	// Every simulated value is inside its metric's range, so it would pass
	// the device endpoint's validation too.
	for metric, id := range ids {
		spec, _ := metric.Spec()
		r, err := f.sensors.Readings(ctx, id, 48)
		if err != nil {
			t.Fatalf("readings: %v", err)
		}
		for _, rd := range r.Readings {
			if rd.Value < spec.Min || rd.Value > spec.Max {
				t.Fatalf("%s value %v outside %v..%v", metric, rd.Value, spec.Min, spec.Max)
			}
		}
	}
}

func TestTickSkipsOtherTypesAndUnconfiguredDevices(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	unconfigured, err := f.entities.Create(ctx, model.EntityInput{
		Name: "Bare sensor", Type: model.TypeIoTDevice, Status: model.StatusActive,
		Latitude: new(float64), Longitude: new(float64),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// A configured device whose type later changed to vehicle.
	changed := f.device(t, model.TypeIoTDevice, model.MetricTemperature)
	if _, err := f.db.Exec(`UPDATE entities SET type = 'vehicle' WHERE id = ?`, changed); err != nil {
		t.Fatalf("change type: %v", err)
	}

	if err := f.sim.Tick(ctx); err != nil {
		t.Fatalf("tick: %v", err)
	}
	for _, id := range []string{unconfigured.ID, changed} {
		if got := f.count(t, id); got != 0 {
			t.Errorf("%s got %d simulated readings, want 0", id, got)
		}
	}
}

func TestTickPrunesOldReadings(t *testing.T) {
	f := newFixture(t)
	id := f.device(t, model.TypeIoTDevice, model.MetricTemperature)
	old := time.Now().UTC().Add(-model.ReadingRetention - time.Hour).Format(time.RFC3339)
	if _, err := f.db.Exec(`INSERT INTO sensor_readings (entity_id, metric, value, recorded_at, received_at)
		VALUES (?, 'temperature', 20, ?, ?)`, id, old, old); err != nil {
		t.Fatalf("insert old reading: %v", err)
	}

	// The first tick prunes before simulating.
	if err := f.sim.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM sensor_readings WHERE recorded_at = ?`, old).Scan(&n); err != nil || n != 0 {
		t.Errorf("old readings left = %d, %v", n, err)
	}
}

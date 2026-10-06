package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

const iotBody = `{"name": "Flood Sensor", "type": "iot_device", "status": "active", "latitude": -6.1, "longitude": 106.8}`

// sensorApp returns an app, an admin cookie and an IoT device id.
func sensorApp(t *testing.T) (testApp, *http.Cookie, string) {
	t.Helper()
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	rec := do(t, app.router, http.MethodPost, "/api/entities", iotBody, admin)
	expectStatus(t, rec, http.StatusCreated)
	return app, admin, decodeEntity(t, rec).ID
}

func setMetric(t *testing.T, app testApp, admin *http.Cookie, id string, metric model.Metric) {
	t.Helper()
	rec := do(t, app.router, http.MethodPut, "/api/entities/"+id+"/sensor", fmt.Sprintf(`{"metric": %q}`, metric), admin)
	expectStatus(t, rec, http.StatusOK)
}

func createKey(t *testing.T, app testApp, admin *http.Cookie, id string) string {
	t.Helper()
	rec := do(t, app.router, http.MethodPost, "/api/entities/"+id+"/sensor/key", "", admin)
	expectStatus(t, rec, http.StatusCreated)
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	key := decode[struct{ Data model.DeviceKey }](t, rec).Data.APIKey
	if !strings.HasPrefix(key, "gem_") || len(key) < 40 {
		t.Fatalf("unexpected key %q", key)
	}
	return key
}

// send posts a reading as a device would, with an optional bearer key.
func send(t *testing.T, app testApp, id, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/devices/"+id+"/readings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	rec := httptest.NewRecorder()
	app.router.ServeHTTP(rec, req)
	return rec
}

func readings(t *testing.T, app testApp, cookie *http.Cookie, id string) *model.Readings {
	t.Helper()
	rec := do(t, app.router, http.MethodGet, "/api/entities/"+id+"/readings", "", cookie)
	expectStatus(t, rec, http.StatusOK)
	return decode[struct{ Data *model.Readings }](t, rec).Data
}

func TestMetaMetrics(t *testing.T) {
	rec := do(t, newServer(t), http.MethodGet, "/api/meta", "")
	expectStatus(t, rec, http.StatusOK)
	got := decode[struct {
		Data struct{ Metrics []model.MetricSpec }
	}](t, rec).Data.Metrics
	if len(got) != len(model.Metrics) || got[0].ID != model.MetricTemperature || got[0].Unit != "°C" || got[0].Min != -50 {
		t.Errorf("metrics = %+v", got)
	}
}

func TestSensorConfigAndKey(t *testing.T) {
	app, admin, id := sensorApp(t)
	path := "/api/entities/" + id + "/sensor"

	rec := do(t, app.router, http.MethodGet, path, "", admin)
	expectStatus(t, rec, http.StatusOK)
	if rec.Body.String() != "{\"data\":null}\n" {
		t.Errorf("empty sensor = %s", rec.Body)
	}

	// A key needs a metric first.
	got := expectError(t, do(t, app.router, http.MethodPost, path+"/key", "", admin), http.StatusBadRequest, "invalid_request")
	if got.Message != "choose the sensor metric first" {
		t.Errorf("message = %q", got.Message)
	}

	got = expectError(t, do(t, app.router, http.MethodPut, path, `{"metric": "humidity"}`, admin),
		http.StatusUnprocessableEntity, "validation_failed")
	if !strings.HasPrefix(got.Fields["metric"], "must be one of: temperature") {
		t.Errorf("fields = %v", got.Fields)
	}

	setMetric(t, app, admin, id, model.MetricWaterLevel)
	createKey(t, app, admin, id)

	rec = do(t, app.router, http.MethodGet, path, "", admin)
	cfg := decode[struct{ Data model.SensorConfig }](t, rec).Data
	if cfg.Metric != model.MetricWaterLevel || !cfg.HasAPIKey || cfg.KeyCreatedAt == nil {
		t.Errorf("config = %+v", cfg)
	}
	if strings.Contains(rec.Body.String(), "hash") || strings.Contains(rec.Body.String(), "gem_") {
		t.Errorf("config leaks the key: %s", rec.Body)
	}

	// Changing the metric keeps the key.
	setMetric(t, app, admin, id, model.MetricTemperature)
	rec = do(t, app.router, http.MethodGet, path, "", admin)
	if cfg := decode[struct{ Data model.SensorConfig }](t, rec).Data; !cfg.HasAPIKey {
		t.Error("changing the metric dropped the key")
	}

	expectStatus(t, do(t, app.router, http.MethodDelete, path, "", admin), http.StatusNoContent)
	rec = do(t, app.router, http.MethodGet, path, "", admin)
	if rec.Body.String() != "{\"data\":null}\n" {
		t.Errorf("after delete = %s", rec.Body)
	}
}

func TestDeviceReadings(t *testing.T) {
	app, admin, id := sensorApp(t)
	setMetric(t, app, admin, id, model.MetricTemperature)
	key := createKey(t, app, admin, id)

	if r := readings(t, app, admin, id); len(r.Readings) != 0 || r.Latest != nil || r.Metric.ID != model.MetricTemperature {
		t.Errorf("before any reading: %+v", r)
	}

	rec := send(t, app, id, key, `{"value": 27.5}`)
	expectStatus(t, rec, http.StatusCreated)
	first := decode[struct{ Data model.Reading }](t, rec).Data
	if first.Value != 27.5 || time.Since(first.RecordedAt) > time.Minute {
		t.Errorf("reading = %+v", first)
	}

	earlier := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	expectStatus(t, send(t, app, id, key, fmt.Sprintf(`{"value": -50, "recorded_at": %q}`, earlier.Format(time.RFC3339))), http.StatusCreated)

	r := readings(t, app, admin, id)
	if len(r.Readings) != 2 || !r.Readings[0].RecordedAt.Equal(earlier) || r.Readings[1].Value != 27.5 {
		t.Errorf("readings, oldest first = %+v", r.Readings)
	}
	if r.Latest == nil || r.Latest.Value != 27.5 {
		t.Errorf("latest = %+v", r.Latest)
	}

	// Validation applies to devices too.
	got := expectError(t, send(t, app, id, key, `{"value": -50.1}`), http.StatusUnprocessableEntity, "validation_failed")
	if got.Fields["value"] != "must be between -50 and 80" {
		t.Errorf("fields = %v", got.Fields)
	}
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	expectError(t, send(t, app, id, key, fmt.Sprintf(`{"value": 1, "recorded_at": %q}`, future)),
		http.StatusUnprocessableEntity, "validation_failed")
	expectError(t, send(t, app, id, key, `{"value": "hot"}`), http.StatusUnprocessableEntity, "validation_failed")
	expectError(t, send(t, app, id, key, `{"value": 1, "extra": true}`), http.StatusBadRequest, "invalid_json")

	// Readings of a previous metric are not shown after switching.
	setMetric(t, app, admin, id, model.MetricWindSpeed)
	if r := readings(t, app, admin, id); len(r.Readings) != 0 || r.Latest != nil || r.Metric.ID != model.MetricWindSpeed {
		t.Errorf("after switching metric: %+v", r)
	}
}

func TestDeviceKeyRejected(t *testing.T) {
	app, admin, id := sensorApp(t)
	setMetric(t, app, admin, id, model.MetricTemperature)
	oldKey := createKey(t, app, admin, id)

	// A second device with its own key.
	rec := do(t, app.router, http.MethodPost, "/api/entities", iotBody, admin)
	otherID := decodeEntity(t, rec).ID
	setMetric(t, app, admin, otherID, model.MetricTemperature)
	otherKey := createKey(t, app, admin, otherID)

	newKey := createKey(t, app, admin, id) // replaces oldKey
	cases := map[string]string{
		"no key":                    "",
		"wrong key":                 "gem_not-a-real-key",
		"another device's key":      otherKey,
		"key replaced by a new one": oldKey,
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			got := expectError(t, send(t, app, id, key, `{"value": 1}`), http.StatusUnauthorized, "unauthorized")
			if got.Message != "invalid device key" {
				t.Errorf("message = %q", got.Message)
			}
		})
	}
	// The key is checked before the body: a bad body with a bad key is still 401.
	expectError(t, send(t, app, id, "", `{not json`), http.StatusUnauthorized, "unauthorized")
	// A session cookie is not a device key.
	req := httptest.NewRequest(http.MethodPost, "/api/devices/"+id+"/readings", strings.NewReader(`{"value": 1}`))
	req.AddCookie(admin)
	rec = httptest.NewRecorder()
	app.router.ServeHTTP(rec, req)
	expectError(t, rec, http.StatusUnauthorized, "unauthorized")

	expectStatus(t, send(t, app, id, newKey, `{"value": 1}`), http.StatusCreated)

	// After the type changes away from iot_device, the key stops working.
	vehicle := `{"name": "Flood Sensor", "type": "vehicle", "status": "active", "latitude": -6.1, "longitude": 106.8}`
	expectStatus(t, do(t, app.router, http.MethodPut, "/api/entities/"+id, vehicle, admin), http.StatusOK)
	expectError(t, send(t, app, id, newKey, `{"value": 1}`), http.StatusUnauthorized, "unauthorized")
}

func TestSensorNotSupportedOrMissing(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	rec := do(t, app.router, http.MethodPost, "/api/entities", facilityBody, admin)
	facility := "/api/entities/" + decodeEntity(t, rec).ID

	for _, rt := range []struct{ method, path string }{
		{http.MethodGet, facility + "/sensor"}, {http.MethodPut, facility + "/sensor"},
		{http.MethodPost, facility + "/sensor/key"}, {http.MethodGet, facility + "/readings"},
	} {
		got := expectError(t, do(t, app.router, rt.method, rt.path, `{"metric": "temperature"}`, admin),
			http.StatusBadRequest, "invalid_request")
		if got.Message != "this entity type has no sensor data" {
			t.Errorf("%s %s: message = %q", rt.method, rt.path, got.Message)
		}
	}
	expectError(t, do(t, app.router, http.MethodGet, "/api/entities/"+missingID+"/sensor", "", admin), http.StatusNotFound, "not_found")
	// A device that does not exist has no valid key either.
	expectError(t, send(t, app, missingID, "gem_x", `{"value": 1}`), http.StatusUnauthorized, "unauthorized")
}

func TestReadingsHoursParam(t *testing.T) {
	app, admin, id := sensorApp(t)
	setMetric(t, app, admin, id, model.MetricTemperature)
	key := createKey(t, app, admin, id)
	threeHoursAgo := time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339)
	expectStatus(t, send(t, app, id, key, fmt.Sprintf(`{"value": 20, "recorded_at": %q}`, threeHoursAgo)), http.StatusCreated)

	get := func(hours string) *httptest.ResponseRecorder {
		return do(t, app.router, http.MethodGet, "/api/entities/"+id+"/readings?hours="+hours, "", admin)
	}
	rec := get("2")
	expectStatus(t, rec, http.StatusOK)
	if r := decode[struct{ Data model.Readings }](t, rec).Data; len(r.Readings) != 0 || r.Latest == nil {
		t.Errorf("hours=2: %+v (latest is reported even when outside the window)", r)
	}
	rec = get("4")
	if r := decode[struct{ Data model.Readings }](t, rec).Data; len(r.Readings) != 1 {
		t.Errorf("hours=4: %+v", r)
	}
	for _, bad := range []string{"0", "169", "abc", "1.5"} {
		got := expectError(t, get(bad), http.StatusUnprocessableEntity, "validation_failed")
		if got.Fields["hours"] == "" {
			t.Errorf("hours=%s: fields = %v", bad, got.Fields)
		}
	}
}

func TestSensorAccessControl(t *testing.T) {
	app, admin, id := sensorApp(t)
	user := register(t, app, "budi@example.com", "rahasia123")
	base := "/api/entities/" + id

	for _, rt := range []struct{ method, path, body string }{
		{http.MethodGet, base + "/sensor", ""}, {http.MethodPut, base + "/sensor", `{"metric": "temperature"}`},
		{http.MethodDelete, base + "/sensor", ""}, {http.MethodPost, base + "/sensor/key", ""},
		{http.MethodGet, base + "/readings", ""},
	} {
		expectError(t, do(t, app.router, rt.method, rt.path, rt.body), http.StatusUnauthorized, "unauthorized")
	}

	expectError(t, do(t, app.router, http.MethodPut, base+"/sensor", `{"metric": "temperature"}`, user), http.StatusForbidden, "forbidden")
	setMetric(t, app, admin, id, model.MetricTemperature)
	expectError(t, do(t, app.router, http.MethodPost, base+"/sensor/key", "", user), http.StatusForbidden, "forbidden")
	expectError(t, do(t, app.router, http.MethodDelete, base+"/sensor", "", user), http.StatusForbidden, "forbidden")
	expectStatus(t, do(t, app.router, http.MethodGet, base+"/sensor", "", user), http.StatusOK)
	expectStatus(t, do(t, app.router, http.MethodGet, base+"/readings", "", user), http.StatusOK)
}

func TestDeletingEntityDeletesReadings(t *testing.T) {
	app, admin, id := sensorApp(t)
	setMetric(t, app, admin, id, model.MetricTemperature)
	expectStatus(t, send(t, app, id, createKey(t, app, admin, id), `{"value": 1}`), http.StatusCreated)
	expectStatus(t, do(t, app.router, http.MethodDelete, "/api/entities/"+id, "", admin), http.StatusNoContent)

	for _, table := range []string{"sensor_configs", "sensor_readings"} {
		var n int
		if err := app.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE entity_id = ?`, id).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s rows left = %d, %v", table, n, err)
		}
	}
}

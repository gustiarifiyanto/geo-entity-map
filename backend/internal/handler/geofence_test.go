package handler_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"testing"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

// vehicleAt creates a vehicle at the given position and returns its id.
func vehicleAt(t *testing.T, app testApp, admin *http.Cookie, lat, lng float64) string {
	t.Helper()
	body := fmt.Sprintf(`{"name": "Truck", "type": "vehicle", "status": "active", "latitude": %v, "longitude": %v}`, lat, lng)
	rec := do(t, app.router, http.MethodPost, "/api/entities", body, admin)
	expectStatus(t, rec, http.StatusCreated)
	return decodeEntity(t, rec).ID
}

func zoneBody(lat, lng, radius float64) string {
	return fmt.Sprintf(`{"center_latitude": %v, "center_longitude": %v, "radius_m": %v}`, lat, lng, radius)
}

func decodeGeofence(t *testing.T, body []byte) *model.Geofence {
	t.Helper()
	var rec struct{ Data *model.Geofence }
	if err := json.Unmarshal(body, &rec); err != nil {
		t.Fatalf("decode geofence %s: %v", body, err)
	}
	return rec.Data
}

func TestGeofenceLifecycle(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	// The vehicle stands at Monas; its zone is centered there with 5 km.
	id := vehicleAt(t, app, admin, -6.175392, 106.827153)
	path := "/api/entities/" + id + "/geofence"

	rec := do(t, app.router, http.MethodGet, path, "", admin)
	expectStatus(t, rec, http.StatusOK)
	if rec.Body.String() != "{\"data\":null}\n" {
		t.Errorf("no zone yet = %s", rec.Body)
	}

	rec = do(t, app.router, http.MethodPut, path, zoneBody(-6.175392, 106.827153, 5000), admin)
	expectStatus(t, rec, http.StatusOK)
	g := decodeGeofence(t, rec.Body.Bytes())
	if g.EntityID != id || g.RadiusM != 5000 || g.DistanceM != 0 || !g.Inside || g.UpdatedAt.IsZero() {
		t.Errorf("after PUT: %+v", g)
	}

	// Moving the vehicle to Bundaran HI (≈2.2 km) keeps it inside...
	expectStatus(t, do(t, app.router, http.MethodPatch, "/api/entities/"+id+"/location",
		`{"latitude": -6.195016, "longitude": 106.822970}`, admin), http.StatusOK)
	g = decodeGeofence(t, do(t, app.router, http.MethodGet, path, "", admin).Body.Bytes())
	if !g.Inside || g.DistanceM < 2200 || g.DistanceM > 2260 {
		t.Errorf("at Bundaran HI: %+v", g)
	}

	// ...and moving it to Bandung (≈120 km) is allowed but outside.
	expectStatus(t, do(t, app.router, http.MethodPatch, "/api/entities/"+id+"/location",
		`{"latitude": -6.9175, "longitude": 107.6191}`, admin), http.StatusOK)
	g = decodeGeofence(t, do(t, app.router, http.MethodGet, path, "", admin).Body.Bytes())
	if g.Inside || g.DistanceM < 100_000 {
		t.Errorf("in Bandung: %+v", g)
	}

	expectStatus(t, do(t, app.router, http.MethodDelete, path, "", admin), http.StatusNoContent)
	expectStatus(t, do(t, app.router, http.MethodDelete, path, "", admin), http.StatusNoContent)
	if rec := do(t, app.router, http.MethodGet, path, "", admin); rec.Body.String() != "{\"data\":null}\n" {
		t.Errorf("after DELETE = %s", rec.Body)
	}
}

func TestGeofenceValidation(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	path := "/api/entities/" + vehicleAt(t, app, admin, -6.2, 106.8) + "/geofence"

	tests := []struct {
		name, body string
		fields     map[string]string
	}{
		{"missing", `{}`, map[string]string{
			"center_latitude": "is required", "center_longitude": "is required", "radius_m": "is required",
		}},
		{"radius too small", zoneBody(-6.2, 106.8, 99.9), map[string]string{"radius_m": "must be between 100 and 50000"}},
		{"radius too large", zoneBody(-6.2, 106.8, 50000.1), map[string]string{"radius_m": "must be between 100 and 50000"}},
		{"center out of range", zoneBody(90.0001, 180.0001, 1000), map[string]string{
			"center_latitude": "must be between -90 and 90", "center_longitude": "must be between -180 and 180",
		}},
		{"wrong type", `{"center_latitude": "north", "center_longitude": 1, "radius_m": 1000}`,
			map[string]string{"center_latitude": "must be a number"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := expectError(t, do(t, app.router, http.MethodPut, path, tc.body, admin),
				http.StatusUnprocessableEntity, "validation_failed")
			if !maps.Equal(got.Fields, tc.fields) {
				t.Errorf("fields = %v, want %v", got.Fields, tc.fields)
			}
		})
	}
	expectError(t, do(t, app.router, http.MethodPut, path, `{"radius_m": 1000, "shape": "square"}`, admin),
		http.StatusBadRequest, "invalid_json")
}

func TestGeofenceNotSupportedOrMissing(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	rec := do(t, app.router, http.MethodPost, "/api/entities", facilityBody, admin)
	facilityPath := "/api/entities/" + decodeEntity(t, rec).ID + "/geofence"
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		got := expectError(t, do(t, app.router, method, facilityPath, zoneBody(0, 0, 1000), admin),
			http.StatusBadRequest, "invalid_request")
		if got.Message != "this entity type has no operating zone" {
			t.Errorf("%s: message = %q", method, got.Message)
		}
	}
	for _, id := range []string{missingID, "not-a-uuid"} {
		expectError(t, do(t, app.router, http.MethodGet, "/api/entities/"+id+"/geofence", "", admin),
			http.StatusNotFound, "not_found")
	}
}

func TestGeofenceListAndCascade(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	center := func(id string) {
		expectStatus(t, do(t, app.router, http.MethodPut, "/api/entities/"+id+"/geofence",
			zoneBody(-6.175392, 106.827153, 1000), admin), http.StatusOK)
	}
	inside := vehicleAt(t, app, admin, -6.175392, 106.827153)      // at the center
	slightlyOut := vehicleAt(t, app, admin, -6.195016, 106.822970) // ≈2.2 km, 1.2 km past the edge
	farOut := vehicleAt(t, app, admin, -6.9175, 107.6191)          // Bandung
	for _, id := range []string{inside, slightlyOut, farOut} {
		center(id)
	}

	list := func() []model.Geofence {
		rec := do(t, app.router, http.MethodGet, "/api/geofences", "", admin)
		expectStatus(t, rec, http.StatusOK)
		return decode[struct{ Data []model.Geofence }](t, rec).Data
	}
	got := list()
	if len(got) != 3 || got[0].EntityID != farOut || got[1].EntityID != slightlyOut || got[2].EntityID != inside ||
		got[0].Inside || got[1].Inside || !got[2].Inside {
		t.Errorf("order (outside first, farthest first) = %+v", got)
	}

	// A type change hides the zone; deleting the entity deletes it.
	facility := `{"name": "Truck", "type": "facility", "status": "active", "latitude": -6.2, "longitude": 106.8}`
	expectStatus(t, do(t, app.router, http.MethodPut, "/api/entities/"+inside, facility, admin), http.StatusOK)
	expectStatus(t, do(t, app.router, http.MethodDelete, "/api/entities/"+farOut, "", admin), http.StatusNoContent)
	if got := list(); len(got) != 1 || got[0].EntityID != slightlyOut {
		t.Errorf("after type change and delete = %+v", got)
	}
	var n int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM geofences WHERE entity_id = ?`, farOut).Scan(&n); err != nil || n != 0 {
		t.Errorf("zone rows left for the deleted entity = %d, %v", n, err)
	}
}

func TestGeofenceAccessControl(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	user := register(t, app, "budi@example.com", "rahasia123")
	path := "/api/entities/" + vehicleAt(t, app, admin, -6.2, 106.8) + "/geofence"
	body := zoneBody(-6.2, 106.8, 1000)

	for _, rt := range []struct{ method, path, body string }{
		{http.MethodGet, path, ""}, {http.MethodPut, path, body}, {http.MethodDelete, path, ""}, {http.MethodGet, "/api/geofences", ""},
	} {
		expectError(t, do(t, app.router, rt.method, rt.path, rt.body), http.StatusUnauthorized, "unauthorized")
	}
	expectError(t, do(t, app.router, http.MethodPut, path, body, user), http.StatusForbidden, "forbidden")
	expectError(t, do(t, app.router, http.MethodDelete, path, "", user), http.StatusForbidden, "forbidden")
	expectStatus(t, do(t, app.router, http.MethodGet, path, "", user), http.StatusOK)
	expectStatus(t, do(t, app.router, http.MethodGet, "/api/geofences", "", user), http.StatusOK)
	expectStatus(t, do(t, app.router, http.MethodPut, path, body, admin), http.StatusOK)
}

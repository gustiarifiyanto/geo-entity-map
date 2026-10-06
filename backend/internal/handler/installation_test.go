package handler_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

const facilityBody = `{
	"name": "Depot",
	"type": "facility",
	"status": "active",
	"latitude": -6.2,
	"longitude": 106.8
}`

// day returns today's date (UTC, as the test app uses) shifted by n days.
func day(n int) string {
	return time.Now().UTC().AddDate(0, 0, n).Format(model.DateLayout)
}

func installationBody(start, target string, completed *string) string {
	c := "null"
	if completed != nil {
		c = fmt.Sprintf("%q", *completed)
	}
	return fmt.Sprintf(`{"started_on": %q, "target_on": %q, "completed_on": %s}`, start, target, c)
}

func decodeInstallation(t *testing.T, body []byte) *model.Installation {
	t.Helper()
	rec := struct{ Data *model.Installation }{}
	if err := json.Unmarshal(body, &rec); err != nil {
		t.Fatalf("decode installation %s: %v", body, err)
	}
	return rec.Data
}

func createFacility(t *testing.T, app testApp, admin *http.Cookie) string {
	t.Helper()
	rec := do(t, app.router, http.MethodPost, "/api/entities", facilityBody, admin)
	expectStatus(t, rec, http.StatusCreated)
	return decodeEntity(t, rec).ID
}

func TestMetaCapabilities(t *testing.T) {
	rec := do(t, newServer(t), http.MethodGet, "/api/meta", "")
	expectStatus(t, rec, http.StatusOK)
	got := decode[struct {
		Data struct {
			Capabilities map[string][]string `json:"capabilities"`
		}
	}](t, rec).Data.Capabilities
	if !slices.Equal(got["facility"], []string{"installation"}) || len(got) != len(model.TypeCapabilities) {
		t.Errorf("capabilities = %v", got)
	}
}

func TestInstallationLifecycle(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	id := createFacility(t, app, admin)
	path := "/api/entities/" + id + "/installation"

	// Not set yet: data is null, not 404.
	rec := do(t, app.router, http.MethodGet, path, "", admin)
	expectStatus(t, rec, http.StatusOK)
	if got := rec.Body.String(); got != "{\"data\":null}\n" {
		t.Errorf("empty installation body = %q", got)
	}

	// Started 5 days ago, target was 2 days ago, not done: overdue by 2.
	rec = do(t, app.router, http.MethodPut, path, installationBody(day(-5), day(-2), nil), admin)
	expectStatus(t, rec, http.StatusOK)
	inst := decodeInstallation(t, rec.Body.Bytes())
	if inst.EntityID != id || inst.Status != model.InstallationOverdue || inst.PlannedDays != 3 ||
		inst.ElapsedDays != 5 || inst.DaysLate != 2 || inst.CompletedOn != nil || inst.UpdatedAt.IsZero() {
		t.Errorf("after first PUT: %+v", inst)
	}

	// Completing it today makes it late, and PUT replaces the whole schedule.
	today := day(0)
	rec = do(t, app.router, http.MethodPut, path, installationBody(day(-5), day(-2), &today), admin)
	expectStatus(t, rec, http.StatusOK)
	if inst := decodeInstallation(t, rec.Body.Bytes()); inst.Status != model.InstallationCompletedLate || inst.DaysLate != 2 {
		t.Errorf("after completing: %+v", inst)
	}

	rec = do(t, app.router, http.MethodGet, path, "", admin)
	expectStatus(t, rec, http.StatusOK)
	if inst := decodeInstallation(t, rec.Body.Bytes()); inst == nil || inst.CompletedOn == nil || *inst.CompletedOn != today {
		t.Errorf("GET after PUT: %+v", inst)
	}

	expectStatus(t, do(t, app.router, http.MethodDelete, path, "", admin), http.StatusNoContent)
	// Deleting again is fine.
	expectStatus(t, do(t, app.router, http.MethodDelete, path, "", admin), http.StatusNoContent)
	rec = do(t, app.router, http.MethodGet, path, "", admin)
	if decodeInstallation(t, rec.Body.Bytes()) != nil {
		t.Error("installation still there after DELETE")
	}
}

func TestInstallationValidation(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	path := "/api/entities/" + createFacility(t, app, admin) + "/installation"
	tomorrow := day(1)

	tests := []struct {
		name   string
		body   string
		fields map[string]string
	}{
		{"missing dates", `{}`, map[string]string{"started_on": "is required", "target_on": "is required"}},
		{"bad format", installationBody("2026/10/01", "2026-02-30", nil), map[string]string{
			"started_on": "must be a date (YYYY-MM-DD)", "target_on": "must be a date (YYYY-MM-DD)",
		}},
		{"target before start", installationBody(day(0), day(-1), nil), map[string]string{
			"target_on": "must be on or after the start date",
		}},
		{"completed tomorrow", installationBody(day(-3), day(3), &tomorrow), map[string]string{
			"completed_on": "cannot be in the future",
		}},
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

	expectError(t, do(t, app.router, http.MethodPut, path, `{"started_on": "2026-10-01", "extra": 1}`, admin),
		http.StatusBadRequest, "invalid_json")
}

func TestInstallationNotSupportedOrMissing(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)

	// validBody is a vehicle: no installation capability.
	rec := do(t, app.router, http.MethodPost, "/api/entities", validBody, admin)
	expectStatus(t, rec, http.StatusCreated)
	vehiclePath := "/api/entities/" + decodeEntity(t, rec).ID + "/installation"
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		got := expectError(t, do(t, app.router, method, vehiclePath, installationBody(day(0), day(1), nil), admin),
			http.StatusBadRequest, "invalid_request")
		if got.Message != "this entity type has no installation data" {
			t.Errorf("%s: message = %q", method, got.Message)
		}
	}

	for _, id := range []string{missingID, "not-a-uuid"} {
		expectError(t, do(t, app.router, http.MethodGet, "/api/entities/"+id+"/installation", "", admin),
			http.StatusNotFound, "not_found")
	}
}

func TestInstallationListAndCascade(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	onTime, lateBy1, lateBy4 := createFacility(t, app, admin), createFacility(t, app, admin), createFacility(t, app, admin)
	put := func(id, start, target string) {
		expectStatus(t, do(t, app.router, http.MethodPut, "/api/entities/"+id+"/installation",
			installationBody(start, target, nil), admin), http.StatusOK)
	}
	put(onTime, day(-1), day(10))
	put(lateBy1, day(-9), day(-1))
	put(lateBy4, day(-9), day(-4))

	list := func() []model.Installation {
		rec := do(t, app.router, http.MethodGet, "/api/installations", "", admin)
		expectStatus(t, rec, http.StatusOK)
		return decode[struct{ Data []model.Installation }](t, rec).Data
	}
	got := list()
	var order []string
	for _, inst := range got {
		order = append(order, inst.EntityID)
	}
	if !slices.Equal(order, []string{lateBy4, lateBy1, onTime}) {
		t.Errorf("order = %v, want most overdue first, then the rest", order)
	}

	// Changing the type away from facility hides its schedule from the list.
	vehicle := `{"name": "Depot", "type": "vehicle", "status": "active", "latitude": -6.2, "longitude": 106.8}`
	expectStatus(t, do(t, app.router, http.MethodPut, "/api/entities/"+onTime, vehicle, admin), http.StatusOK)
	// Deleting an entity deletes its schedule.
	expectStatus(t, do(t, app.router, http.MethodDelete, "/api/entities/"+lateBy1, "", admin), http.StatusNoContent)

	got = list()
	if len(got) != 1 || got[0].EntityID != lateBy4 {
		t.Errorf("list after type change and delete = %+v, want only %s", got, lateBy4)
	}
	var rows int
	if err := app.db.QueryRow(`SELECT COUNT(*) FROM facility_installations WHERE entity_id = ?`, lateBy1).Scan(&rows); err != nil || rows != 0 {
		t.Errorf("rows left for the deleted entity = %d, %v", rows, err)
	}
}

func TestInstallationAccessControl(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	user := register(t, app, "budi@example.com", "rahasia123")
	path := "/api/entities/" + createFacility(t, app, admin) + "/installation"
	body := installationBody(day(0), day(5), nil)

	for _, rt := range []struct{ method, path, body string }{
		{http.MethodGet, path, ""}, {http.MethodPut, path, body}, {http.MethodDelete, path, ""}, {http.MethodGet, "/api/installations", ""},
	} {
		expectError(t, do(t, app.router, rt.method, rt.path, rt.body), http.StatusUnauthorized, "unauthorized")
	}

	expectError(t, do(t, app.router, http.MethodPut, path, body, user), http.StatusForbidden, "forbidden")
	expectError(t, do(t, app.router, http.MethodDelete, path, "", user), http.StatusForbidden, "forbidden")
	expectStatus(t, do(t, app.router, http.MethodGet, path, "", user), http.StatusOK)
	expectStatus(t, do(t, app.router, http.MethodGet, "/api/installations", "", user), http.StatusOK)

	// The admin can.
	expectStatus(t, do(t, app.router, http.MethodPut, path, body, admin), http.StatusOK)
	expectStatus(t, do(t, app.router, http.MethodDelete, path, "", admin), http.StatusNoContent)
}

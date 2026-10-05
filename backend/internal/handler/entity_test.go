package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/database"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/handler"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

const validBody = `{
	"name": "Truck 1",
	"type": "vehicle",
	"status": "active",
	"latitude": -6.2,
	"longitude": 106.8,
	"description": "A truck",
	"attributes": {"plate": "B 1"}
}`

const missingID = "00000000-0000-4000-8000-000000000000"

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

func newServer(t *testing.T) http.Handler {
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
	val, err := validation.New()
	if err != nil {
		t.Fatalf("validator: %v", err)
	}
	return handler.NewRouter(service.NewEntityService(repository.NewEntityRepository(db)), val)
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, rec.Body)
	}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body, err)
	}
	return v
}

func decodeEntity(t *testing.T, rec *httptest.ResponseRecorder) model.Entity {
	t.Helper()
	return decode[struct{ Data model.Entity }](t, rec).Data
}

// isNull reports whether a decoded attributes value is absent or JSON null.
func isNull(raw json.RawMessage) bool {
	return len(raw) == 0 || string(raw) == "null"
}

func createEntity(t *testing.T, h http.Handler, body string) model.Entity {
	t.Helper()
	rec := do(t, h, http.MethodPost, "/api/entities", body)
	expectStatus(t, rec, http.StatusCreated)
	return decodeEntity(t, rec)
}

func expectError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) errorResponse {
	t.Helper()
	expectStatus(t, rec, status)
	got := decode[errorResponse](t, rec)
	if got.Error != code {
		t.Fatalf("error = %q, want %q", got.Error, code)
	}
	return got
}

func TestMeta(t *testing.T) {
	h := newServer(t)
	rec := do(t, h, http.MethodGet, "/api/meta", "")
	expectStatus(t, rec, http.StatusOK)

	got := decode[struct {
		Data struct {
			Types    []string `json:"types"`
			Statuses []string `json:"statuses"`
		}
	}](t, rec).Data
	if len(got.Types) != len(model.EntityTypes) || len(got.Statuses) != len(model.EntityStatuses) {
		t.Fatalf("meta = %+v, want %d types and %d statuses", got, len(model.EntityTypes), len(model.EntityStatuses))
	}
}

func TestCreate(t *testing.T) {
	h := newServer(t)
	rec := do(t, h, http.MethodPost, "/api/entities", validBody)
	expectStatus(t, rec, http.StatusCreated)

	e := decodeEntity(t, rec)
	if e.ID == "" || e.CreatedAt.IsZero() || !e.CreatedAt.Equal(e.UpdatedAt) {
		t.Errorf("server-generated fields not set: %+v", e)
	}
	if e.Name != "Truck 1" || e.Type != model.TypeVehicle || e.Latitude != -6.2 || e.Longitude != 106.8 {
		t.Errorf("unexpected entity: %+v", e)
	}
	if string(e.Attributes) != `{"plate":"B 1"}` {
		t.Errorf("attributes = %s", e.Attributes)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/entities/"+e.ID {
		t.Errorf("Location = %q", loc)
	}
}

func TestCreateTrimsAndAllowsOptionalFields(t *testing.T) {
	h := newServer(t)
	e := createEntity(t, h, `{"name":"  Pole  ","type":"facility","status":"inactive","latitude":0,"longitude":0}`)
	if e.Name != "Pole" || e.Description != "" || !isNull(e.Attributes) {
		t.Errorf("unexpected entity: %+v (attributes %s)", e, e.Attributes)
	}
}

func TestCreateValidationErrors(t *testing.T) {
	h := newServer(t)
	rec := do(t, h, http.MethodPost, "/api/entities",
		`{"name":" ","type":"rocket","status":"active","latitude":90.0001,"attributes":[1]}`)
	got := expectError(t, rec, http.StatusUnprocessableEntity, "validation_failed")

	want := map[string]string{
		"name":       "is required",
		"type":       "must be one of: vehicle, iot_device, facility",
		"latitude":   "must be between -90 and 90",
		"longitude":  "is required",
		"attributes": "must be a JSON object",
	}
	if !maps.Equal(got.Fields, want) {
		t.Errorf("fields = %v, want %v", got.Fields, want)
	}
}

func TestCreateWrongFieldType(t *testing.T) {
	tests := []struct {
		name, body, field, want string
	}{
		{"string for number", `{"name":"x","type":"vehicle","status":"active","latitude":"north","longitude":1}`,
			"latitude", "must be a number"},
		{"number for string", `{"name":42,"type":"vehicle","status":"active","latitude":1,"longitude":1}`,
			"name", "must be a string"},
		{"number overflows float64", `{"name":"x","type":"vehicle","status":"active","latitude":1e400,"longitude":1}`,
			"latitude", "is out of range"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newServer(t)
			got := expectError(t, do(t, h, http.MethodPost, "/api/entities", tc.body),
				http.StatusUnprocessableEntity, "validation_failed")
			if got.Fields[tc.field] != tc.want {
				t.Errorf("fields = %v, want %s: %q", got.Fields, tc.field, tc.want)
			}
		})
	}
}

func TestCreateInvalidJSON(t *testing.T) {
	tests := map[string]string{
		"malformed":      `{"name":`,
		"empty body":     ``,
		"unknown field":  `{"name":"x","type":"vehicle","status":"active","latitude":1,"longitude":1,"color":"red"}`,
		"not an object":  `[1,2]`,
		"trailing value": validBody + `{}`,
		"NaN literal":    `{"name":"x","type":"vehicle","status":"active","latitude":NaN,"longitude":1}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			h := newServer(t)
			expectError(t, do(t, h, http.MethodPost, "/api/entities", body), http.StatusBadRequest, "invalid_json")
		})
	}
}

func TestList(t *testing.T) {
	h := newServer(t)

	rec := do(t, h, http.MethodGet, "/api/entities", "")
	expectStatus(t, rec, http.StatusOK)
	if body := strings.TrimSpace(rec.Body.String()); body != `{"data":[]}` {
		t.Errorf("empty list body = %s, want {\"data\":[]}", body)
	}

	createEntity(t, h, validBody)
	createEntity(t, h, `{"name":"Sensor","type":"iot_device","status":"maintenance","latitude":1,"longitude":2}`)
	createEntity(t, h, `{"name":"Depot","type":"facility","status":"active","latitude":3,"longitude":4}`)

	tests := []struct {
		query string
		want  int
	}{
		{"", 3},
		{"?type=vehicle", 1},
		{"?status=active", 2},
		{"?type=facility&status=active", 1},
		{"?type=facility&status=maintenance", 0},
	}
	for _, tc := range tests {
		rec := do(t, h, http.MethodGet, "/api/entities"+tc.query, "")
		expectStatus(t, rec, http.StatusOK)
		if got := len(decode[struct{ Data []model.Entity }](t, rec).Data); got != tc.want {
			t.Errorf("GET %q: %d entities, want %d", tc.query, got, tc.want)
		}
	}
}

func TestListInvalidFilter(t *testing.T) {
	h := newServer(t)
	got := expectError(t, do(t, h, http.MethodGet, "/api/entities?type=rocket&status=x", ""),
		http.StatusUnprocessableEntity, "validation_failed")
	if _, ok := got.Fields["type"]; !ok {
		t.Errorf("fields = %v, want type error", got.Fields)
	}
	if _, ok := got.Fields["status"]; !ok {
		t.Errorf("fields = %v, want status error", got.Fields)
	}
}

func TestGet(t *testing.T) {
	h := newServer(t)
	created := createEntity(t, h, validBody)

	rec := do(t, h, http.MethodGet, "/api/entities/"+created.ID, "")
	expectStatus(t, rec, http.StatusOK)
	if got := decodeEntity(t, rec); got.ID != created.ID || got.Name != created.Name {
		t.Errorf("got %+v, want %+v", got, created)
	}
}

func TestNotFound(t *testing.T) {
	h := newServer(t)
	tests := []struct {
		name, method, path, body string
	}{
		{"get missing", http.MethodGet, "/api/entities/" + missingID, ""},
		{"get malformed id", http.MethodGet, "/api/entities/not-a-uuid", ""},
		{"update missing", http.MethodPut, "/api/entities/" + missingID, validBody},
		{"update malformed id", http.MethodPut, "/api/entities/123", validBody},
		{"location missing", http.MethodPatch, "/api/entities/" + missingID + "/location", `{"latitude":1,"longitude":1}`},
		{"delete missing", http.MethodDelete, "/api/entities/" + missingID, ""},
		{"delete malformed id", http.MethodDelete, "/api/entities/abc", ""},
		{"unknown route", http.MethodGet, "/api/nope", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expectError(t, do(t, h, tc.method, tc.path, tc.body), http.StatusNotFound, "not_found")
		})
	}
}

func TestUpdate(t *testing.T) {
	h := newServer(t)
	created := createEntity(t, h, validBody)

	rec := do(t, h, http.MethodPut, "/api/entities/"+created.ID,
		`{"name":"Truck 2","type":"vehicle","status":"maintenance","latitude":10,"longitude":20}`)
	expectStatus(t, rec, http.StatusOK)

	got := decodeEntity(t, rec)
	if got.Name != "Truck 2" || got.Status != model.StatusMaintenance || got.Latitude != 10 || got.Longitude != 20 {
		t.Errorf("unexpected entity: %+v", got)
	}
	// PUT is a full update: omitted optional fields are cleared.
	if got.Description != "" || !isNull(got.Attributes) {
		t.Errorf("optional fields not cleared: description %q, attributes %s", got.Description, got.Attributes)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("created_at changed: %v -> %v", created.CreatedAt, got.CreatedAt)
	}

	// The change is persisted.
	if stored := decodeEntity(t, do(t, h, http.MethodGet, "/api/entities/"+created.ID, "")); stored.Name != "Truck 2" {
		t.Errorf("stored name = %q, want %q", stored.Name, "Truck 2")
	}
}

func TestUpdateValidationError(t *testing.T) {
	h := newServer(t)
	created := createEntity(t, h, validBody)
	got := expectError(t, do(t, h, http.MethodPut, "/api/entities/"+created.ID,
		`{"name":"x","type":"vehicle","status":"active","latitude":1,"longitude":180.0001}`),
		http.StatusUnprocessableEntity, "validation_failed")
	if got.Fields["longitude"] != "must be between -180 and 180" {
		t.Errorf("fields = %v", got.Fields)
	}
}

func TestUpdateLocation(t *testing.T) {
	h := newServer(t)
	created := createEntity(t, h, validBody)

	rec := do(t, h, http.MethodPatch, "/api/entities/"+created.ID+"/location", `{"latitude":-7.25,"longitude":112.75}`)
	expectStatus(t, rec, http.StatusOK)

	got := decodeEntity(t, rec)
	if got.Latitude != -7.25 || got.Longitude != 112.75 {
		t.Errorf("location = (%v, %v), want (-7.25, 112.75)", got.Latitude, got.Longitude)
	}
	// Other fields are untouched.
	if got.Name != created.Name || string(got.Attributes) != string(created.Attributes) {
		t.Errorf("other fields changed: %+v", got)
	}
}

func TestUpdateLocationErrors(t *testing.T) {
	h := newServer(t)
	created := createEntity(t, h, validBody)
	path := "/api/entities/" + created.ID + "/location"

	got := expectError(t, do(t, h, http.MethodPatch, path, `{"latitude":-91,"longitude":200}`),
		http.StatusUnprocessableEntity, "validation_failed")
	if len(got.Fields) != 2 {
		t.Errorf("fields = %v, want latitude and longitude errors", got.Fields)
	}

	expectError(t, do(t, h, http.MethodPatch, path, `{"latitude":1,"longitude":1,"name":"x"}`),
		http.StatusBadRequest, "invalid_json")
}

func TestDelete(t *testing.T) {
	h := newServer(t)
	created := createEntity(t, h, validBody)

	rec := do(t, h, http.MethodDelete, "/api/entities/"+created.ID, "")
	expectStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Errorf("204 body = %q, want empty", rec.Body)
	}

	expectError(t, do(t, h, http.MethodGet, "/api/entities/"+created.ID, ""), http.StatusNotFound, "not_found")
	expectError(t, do(t, h, http.MethodDelete, "/api/entities/"+created.ID, ""), http.StatusNotFound, "not_found")
}

func TestMethodNotAllowed(t *testing.T) {
	h := newServer(t)
	expectError(t, do(t, h, http.MethodDelete, "/api/entities", ""), http.StatusMethodNotAllowed, "method_not_allowed")
}

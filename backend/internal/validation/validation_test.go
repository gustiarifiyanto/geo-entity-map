package validation

import (
	"encoding/json"
	"maps"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

func newValidator(t *testing.T) *Validator {
	t.Helper()
	v, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

func ptr(f float64) *float64 { return &f }

func validEntity() model.EntityInput {
	return model.EntityInput{
		Name:        "Truck 1",
		Type:        model.TypeVehicle,
		Status:      model.StatusActive,
		Latitude:    ptr(-6.2),
		Longitude:   ptr(106.8),
		Description: "A truck",
		Attributes:  json.RawMessage(`{"plate":"B 1"}`),
	}
}

func TestEntity(t *testing.T) {
	tests := []struct {
		name   string
		modify func(in *model.EntityInput)
		want   FieldErrors // nil means valid
	}{
		{"valid", func(*model.EntityInput) {}, nil},
		{"valid without optional fields", func(in *model.EntityInput) {
			in.Description = ""
			in.Attributes = nil
		}, nil},

		// name
		{"name missing", func(in *model.EntityInput) { in.Name = "" },
			FieldErrors{"name": "is required"}},
		{"name only whitespace", func(in *model.EntityInput) { in.Name = "   \t " },
			FieldErrors{"name": "is required"}},
		{"name 100 chars", func(in *model.EntityInput) { in.Name = strings.Repeat("a", 100) }, nil},
		{"name 100 chars with surrounding spaces", func(in *model.EntityInput) {
			in.Name = "  " + strings.Repeat("a", 100) + "  "
		}, nil},
		{"name 100 multibyte chars", func(in *model.EntityInput) { in.Name = strings.Repeat("é", 100) }, nil},
		{"name 101 chars", func(in *model.EntityInput) { in.Name = strings.Repeat("a", 101) },
			FieldErrors{"name": "must be at most 100 characters"}},

		// type
		{"type missing", func(in *model.EntityInput) { in.Type = "" },
			FieldErrors{"type": "is required"}},
		{"type unknown", func(in *model.EntityInput) { in.Type = "spaceship" },
			FieldErrors{"type": "must be one of: vehicle, iot_device, facility"}},
		{"type wrong case", func(in *model.EntityInput) { in.Type = "Vehicle" },
			FieldErrors{"type": "must be one of: vehicle, iot_device, facility"}},

		// status
		{"status missing", func(in *model.EntityInput) { in.Status = "" },
			FieldErrors{"status": "is required"}},
		{"status unknown", func(in *model.EntityInput) { in.Status = "broken" },
			FieldErrors{"status": "must be one of: active, inactive, maintenance"}},

		// latitude
		{"latitude missing", func(in *model.EntityInput) { in.Latitude = nil },
			FieldErrors{"latitude": "is required"}},
		{"latitude zero", func(in *model.EntityInput) { in.Latitude = ptr(0) }, nil},
		{"latitude 90", func(in *model.EntityInput) { in.Latitude = ptr(90) }, nil},
		{"latitude -90", func(in *model.EntityInput) { in.Latitude = ptr(-90) }, nil},
		{"latitude 90.0001", func(in *model.EntityInput) { in.Latitude = ptr(90.0001) },
			FieldErrors{"latitude": "must be between -90 and 90"}},
		{"latitude -90.0001", func(in *model.EntityInput) { in.Latitude = ptr(-90.0001) },
			FieldErrors{"latitude": "must be between -90 and 90"}},
		{"latitude NaN", func(in *model.EntityInput) { in.Latitude = ptr(math.NaN()) },
			FieldErrors{"latitude": "must be between -90 and 90"}},
		{"latitude +Inf", func(in *model.EntityInput) { in.Latitude = ptr(math.Inf(1)) },
			FieldErrors{"latitude": "must be between -90 and 90"}},

		// longitude
		{"longitude missing", func(in *model.EntityInput) { in.Longitude = nil },
			FieldErrors{"longitude": "is required"}},
		{"longitude zero", func(in *model.EntityInput) { in.Longitude = ptr(0) }, nil},
		{"longitude 180", func(in *model.EntityInput) { in.Longitude = ptr(180) }, nil},
		{"longitude -180", func(in *model.EntityInput) { in.Longitude = ptr(-180) }, nil},
		{"longitude 180.0001", func(in *model.EntityInput) { in.Longitude = ptr(180.0001) },
			FieldErrors{"longitude": "must be between -180 and 180"}},
		{"longitude -180.0001", func(in *model.EntityInput) { in.Longitude = ptr(-180.0001) },
			FieldErrors{"longitude": "must be between -180 and 180"}},
		{"longitude -Inf", func(in *model.EntityInput) { in.Longitude = ptr(math.Inf(-1)) },
			FieldErrors{"longitude": "must be between -180 and 180"}},
		{"longitude NaN", func(in *model.EntityInput) { in.Longitude = ptr(math.NaN()) },
			FieldErrors{"longitude": "must be between -180 and 180"}},

		// description
		{"description 500 chars", func(in *model.EntityInput) { in.Description = strings.Repeat("d", 500) }, nil},
		{"description 501 chars", func(in *model.EntityInput) { in.Description = strings.Repeat("d", 501) },
			FieldErrors{"description": "must be at most 500 characters"}},

		// attributes
		{"attributes empty object", func(in *model.EntityInput) { in.Attributes = json.RawMessage(`{}`) }, nil},
		{"attributes nested object", func(in *model.EntityInput) {
			in.Attributes = json.RawMessage(`{"a":{"b":[1,2]},"c":null}`)
		}, nil},
		{"attributes null", func(in *model.EntityInput) { in.Attributes = json.RawMessage(`null`) }, nil},
		{"attributes array", func(in *model.EntityInput) { in.Attributes = json.RawMessage(`[1,2]`) },
			FieldErrors{"attributes": "must be a JSON object"}},
		{"attributes string", func(in *model.EntityInput) { in.Attributes = json.RawMessage(`"x"`) },
			FieldErrors{"attributes": "must be a JSON object"}},
		{"attributes number", func(in *model.EntityInput) { in.Attributes = json.RawMessage(`42`) },
			FieldErrors{"attributes": "must be a JSON object"}},
		{"attributes boolean", func(in *model.EntityInput) { in.Attributes = json.RawMessage(`true`) },
			FieldErrors{"attributes": "must be a JSON object"}},

		// multiple errors are all reported
		{"multiple invalid fields", func(in *model.EntityInput) {
			in.Name = ""
			in.Latitude = ptr(91)
			in.Status = "nope"
		}, FieldErrors{
			"name":     "is required",
			"latitude": "must be between -90 and 90",
			"status":   "must be one of: active, inactive, maintenance",
		}},
	}

	v := newValidator(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := validEntity()
			tc.modify(&in)
			got, err := v.Entity(&in)
			if err != nil {
				t.Fatalf("Entity: unexpected error: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("Entity() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEntityNormalizes(t *testing.T) {
	v := newValidator(t)
	in := validEntity()
	in.Name = "  Truck 1  "
	in.Description = "\n desc \t"
	in.Attributes = json.RawMessage(`null`)

	if got, err := v.Entity(&in); err != nil || got != nil {
		t.Fatalf("Entity() = %v, %v; want nil, nil", got, err)
	}
	if in.Name != "Truck 1" {
		t.Errorf("Name = %q, want trimmed %q", in.Name, "Truck 1")
	}
	if in.Description != "desc" {
		t.Errorf("Description = %q, want trimmed %q", in.Description, "desc")
	}
	if in.Attributes != nil {
		t.Errorf("Attributes = %s, want nil for JSON null", in.Attributes)
	}
}

func TestLocation(t *testing.T) {
	tests := []struct {
		name string
		in   model.LocationInput
		want FieldErrors
	}{
		{"valid", model.LocationInput{Latitude: ptr(-6.2), Longitude: ptr(106.8)}, nil},
		{"zero coordinates", model.LocationInput{Latitude: ptr(0), Longitude: ptr(0)}, nil},
		{"boundaries", model.LocationInput{Latitude: ptr(-90), Longitude: ptr(180)}, nil},
		{"both missing", model.LocationInput{}, FieldErrors{
			"latitude":  "is required",
			"longitude": "is required",
		}},
		{"out of range", model.LocationInput{Latitude: ptr(90.0001), Longitude: ptr(-180.0001)}, FieldErrors{
			"latitude":  "must be between -90 and 90",
			"longitude": "must be between -180 and 180",
		}},
		{"unwrapped Leaflet longitude", model.LocationInput{Latitude: ptr(0), Longitude: ptr(466.8)}, FieldErrors{
			"longitude": "must be between -180 and 180",
		}},
	}

	v := newValidator(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := v.Location(&tc.in)
			if err != nil {
				t.Fatalf("Location: unexpected error: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("Location() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name string
		in   model.RegisterInput
		want FieldErrors
	}{
		{"valid", model.RegisterInput{Email: "budi@example.com", Password: "rahasia123"}, nil},
		{"email is trimmed and lowercased", model.RegisterInput{Email: "  Budi@Example.COM ", Password: "rahasia123"}, nil},
		{"both missing", model.RegisterInput{}, FieldErrors{
			"email":    "is required",
			"password": "is required",
		}},
		{"email only whitespace", model.RegisterInput{Email: "   ", Password: "rahasia123"}, FieldErrors{
			"email": "is required",
		}},
		{"email without @", model.RegisterInput{Email: "budi.example.com", Password: "rahasia123"}, FieldErrors{
			"email": "must be a valid email address",
		}},
		{"email 254 chars", model.RegisterInput{Email: emailOfLength(254), Password: "rahasia123"}, nil},
		{"email 255 chars", model.RegisterInput{Email: emailOfLength(255), Password: "rahasia123"}, FieldErrors{
			"email": "must be at most 254 characters",
		}},
		{"password 7 chars", model.RegisterInput{Email: "budi@example.com", Password: "1234567"}, FieldErrors{
			"password": "must be at least 8 characters",
		}},
		{"password 8 chars", model.RegisterInput{Email: "budi@example.com", Password: "12345678"}, nil},
		{"password of spaces is kept", model.RegisterInput{Email: "budi@example.com", Password: "        "}, nil},
		{"password 72 bytes", model.RegisterInput{Email: "budi@example.com", Password: strings.Repeat("a", 72)}, nil},
		{"password 73 bytes", model.RegisterInput{Email: "budi@example.com", Password: strings.Repeat("a", 73)}, FieldErrors{
			"password": "must be at most 72 bytes",
		}},
		{"password 37 multibyte chars is 74 bytes", model.RegisterInput{Email: "budi@example.com", Password: strings.Repeat("é", 37)}, FieldErrors{
			"password": "must be at most 72 bytes",
		}},
	}

	v := newValidator(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := v.Register(&tc.in)
			if err != nil {
				t.Fatalf("Register: unexpected error: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("Register() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRegisterNormalizes(t *testing.T) {
	in := model.RegisterInput{Email: "  Budi@Example.COM ", Password: "  rahasia  "}
	if _, err := newValidator(t).Register(&in); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if in.Email != "budi@example.com" {
		t.Errorf("Email = %q, want trimmed and lowercased", in.Email)
	}
	if in.Password != "  rahasia  " {
		t.Errorf("Password = %q, want it unchanged", in.Password)
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name string
		in   model.LoginInput
		want FieldErrors
	}{
		{"valid", model.LoginInput{Email: "budi@example.com", Password: "rahasia123"}, nil},
		// Login only checks presence: a short password is a wrong password (401), not a 422.
		{"short password", model.LoginInput{Email: "budi@example.com", Password: "x"}, nil},
		{"both missing", model.LoginInput{}, FieldErrors{
			"email":    "is required",
			"password": "is required",
		}},
		{"email only whitespace", model.LoginInput{Email: " ", Password: "x"}, FieldErrors{
			"email": "is required",
		}},
	}

	v := newValidator(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := v.Login(&tc.in)
			if err != nil {
				t.Fatalf("Login: unexpected error: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("Login() = %v, want %v", got, tc.want)
			}
		})
	}
}

// emailOfLength builds a valid email of exactly n characters (198 <= n <= 260),
// keeping every domain label within the 63-character DNS limit.
func emailOfLength(n int) string {
	local := strings.Repeat("a", 64)
	b, c := strings.Repeat("b", 63), strings.Repeat("c", 63)
	fixed := len(local) + len("@") + len(b) + len(".") + len(c) + len(".") + len(".com")
	return local + "@" + b + "." + c + "." + strings.Repeat("d", n-fixed) + ".com"
}

func strPtr(s string) *string { return &s }

func TestInstallation(t *testing.T) {
	today := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		in   model.InstallationInput
		want FieldErrors
	}{
		{"valid, not completed", model.InstallationInput{StartedOn: "2026-10-01", TargetOn: "2026-10-20"}, nil},
		{"valid, completed today", model.InstallationInput{StartedOn: "2026-10-01", TargetOn: "2026-10-20", CompletedOn: strPtr("2026-10-06")}, nil},
		{"same day start, target and completion", model.InstallationInput{StartedOn: "2026-10-06", TargetOn: "2026-10-06", CompletedOn: strPtr("2026-10-06")}, nil},
		{"start in the future is allowed", model.InstallationInput{StartedOn: "2026-12-01", TargetOn: "2026-12-31"}, nil},
		{"empty completion means not completed", model.InstallationInput{StartedOn: "2026-10-01", TargetOn: "2026-10-20", CompletedOn: strPtr("  ")}, nil},
		{"dates are trimmed", model.InstallationInput{StartedOn: " 2026-10-01 ", TargetOn: "2026-10-20 "}, nil},
		{"both missing", model.InstallationInput{}, FieldErrors{
			"started_on": "is required",
			"target_on":  "is required",
		}},
		{"bad formats", model.InstallationInput{StartedOn: "01-10-2026", TargetOn: "2026-1-5", CompletedOn: strPtr("2026-10-06T00:00:00Z")}, FieldErrors{
			"started_on":   "must be a date (YYYY-MM-DD)",
			"target_on":    "must be a date (YYYY-MM-DD)",
			"completed_on": "must be a date (YYYY-MM-DD)",
		}},
		{"day that does not exist", model.InstallationInput{StartedOn: "2026-02-30", TargetOn: "2026-10-20"}, FieldErrors{
			"started_on": "must be a date (YYYY-MM-DD)",
		}},
		{"target before start", model.InstallationInput{StartedOn: "2026-10-10", TargetOn: "2026-10-09"}, FieldErrors{
			"target_on": "must be on or after the start date",
		}},
		{"completed before start", model.InstallationInput{StartedOn: "2026-10-03", TargetOn: "2026-10-20", CompletedOn: strPtr("2026-10-02")}, FieldErrors{
			"completed_on": "must be on or after the start date",
		}},
		{"completed tomorrow", model.InstallationInput{StartedOn: "2026-10-01", TargetOn: "2026-10-20", CompletedOn: strPtr("2026-10-07")}, FieldErrors{
			"completed_on": "cannot be in the future",
		}},
	}

	v := newValidator(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := v.Installation(&tc.in, today)
			if err != nil {
				t.Fatalf("Installation: unexpected error: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("Installation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSensor(t *testing.T) {
	v := newValidator(t)
	for _, m := range model.Metrics {
		in := model.SensorInput{Metric: m.ID}
		if got, err := v.Sensor(&in); err != nil || got != nil {
			t.Errorf("Sensor(%q) = %v, %v; want valid", m.ID, got, err)
		}
	}
	in := model.SensorInput{Metric: " temperature "}
	if got, _ := v.Sensor(&in); got != nil || in.Metric != model.MetricTemperature {
		t.Errorf("trimmed metric: got %v, metric %q", got, in.Metric)
	}
	for _, bad := range []model.Metric{"", "humidity", "Temperature"} {
		in := model.SensorInput{Metric: bad}
		got, err := v.Sensor(&in)
		if err != nil || got["metric"] == "" {
			t.Errorf("Sensor(%q) = %v, %v; want a metric error", bad, got, err)
		}
	}
}

func TestReading(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	spec, _ := model.MetricTemperature.Spec()
	at := func(d time.Duration) *string { s := now.Add(d).Format(time.RFC3339); return &s }
	tests := []struct {
		name string
		in   model.ReadingInput
		want FieldErrors
	}{
		{"valid", model.ReadingInput{Value: ptr(27.5)}, nil},
		{"lowest value", model.ReadingInput{Value: ptr(-50)}, nil},
		{"highest value", model.ReadingInput{Value: ptr(80)}, nil},
		{"zero", model.ReadingInput{Value: ptr(0)}, nil},
		{"just below range", model.ReadingInput{Value: ptr(-50.1)}, FieldErrors{"value": "must be between -50 and 80"}},
		{"just above range", model.ReadingInput{Value: ptr(80.1)}, FieldErrors{"value": "must be between -50 and 80"}},
		{"missing value", model.ReadingInput{}, FieldErrors{"value": "is required"}},
		{"recorded now", model.ReadingInput{Value: ptr(1), RecordedAt: at(0)}, nil},
		{"recorded 5 min ahead", model.ReadingInput{Value: ptr(1), RecordedAt: at(5 * time.Minute)}, nil},
		{"recorded 5 min 1 s ahead", model.ReadingInput{Value: ptr(1), RecordedAt: at(5*time.Minute + time.Second)},
			FieldErrors{"recorded_at": "cannot be in the future"}},
		{"recorded 7 days ago", model.ReadingInput{Value: ptr(1), RecordedAt: at(-7 * 24 * time.Hour)}, nil},
		{"recorded 7 days 1 s ago", model.ReadingInput{Value: ptr(1), RecordedAt: at(-7*24*time.Hour - time.Second)},
			FieldErrors{"recorded_at": "cannot be older than 7 days"}},
		{"recorded_at with offset", model.ReadingInput{Value: ptr(1), RecordedAt: strPtr("2026-10-06T18:30:00+07:00")}, nil},
		{"recorded_at not RFC3339", model.ReadingInput{Value: ptr(1), RecordedAt: strPtr("2026-10-06 12:00")},
			FieldErrors{"recorded_at": "must be a date-time (RFC3339, e.g. 2026-10-06T08:00:00Z)"}},
	}

	v := newValidator(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := v.Reading(&tc.in, spec, now)
			if err != nil {
				t.Fatalf("Reading: unexpected error: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("Reading() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGeofence(t *testing.T) {
	zone := func(lat, lng, r *float64) model.GeofenceInput {
		return model.GeofenceInput{CenterLatitude: lat, CenterLongitude: lng, RadiusM: r}
	}
	tests := []struct {
		name string
		in   model.GeofenceInput
		want FieldErrors
	}{
		{"valid", zone(ptr(-6.2), ptr(106.8), ptr(5000)), nil},
		{"smallest radius", zone(ptr(0), ptr(0), ptr(100)), nil},
		{"largest radius", zone(ptr(0), ptr(0), ptr(50_000)), nil},
		{"center on the boundaries", zone(ptr(90), ptr(-180), ptr(1000)), nil},
		{"radius 99.9", zone(ptr(0), ptr(0), ptr(99.9)), FieldErrors{"radius_m": "must be between 100 and 50000"}},
		{"radius 50000.1", zone(ptr(0), ptr(0), ptr(50_000.1)), FieldErrors{"radius_m": "must be between 100 and 50000"}},
		{"radius NaN", zone(ptr(0), ptr(0), ptr(math.NaN())), FieldErrors{"radius_m": "must be between 100 and 50000"}},
		{"center out of range", zone(ptr(90.0001), ptr(180.0001), ptr(1000)), FieldErrors{
			"center_latitude":  "must be between -90 and 90",
			"center_longitude": "must be between -180 and 180",
		}},
		{"all missing", model.GeofenceInput{}, FieldErrors{
			"center_latitude":  "is required",
			"center_longitude": "is required",
			"radius_m":         "is required",
		}},
	}
	v := newValidator(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := v.Geofence(&tc.in)
			if err != nil {
				t.Fatalf("Geofence: unexpected error: %v", err)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("Geofence() = %v, want %v", got, tc.want)
			}
		})
	}
}

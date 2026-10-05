package validation

import (
	"encoding/json"
	"maps"
	"math"
	"strings"
	"testing"

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

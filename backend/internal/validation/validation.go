// Package validation validates request inputs and formats the errors as a
// map keyed by JSON field name.
package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

// FieldErrors maps a JSON field name to a human-readable message.
type FieldErrors map[string]string

// Validator validates model inputs.
type Validator struct {
	v *validator.Validate
}

// New returns a Validator with the custom rules registered.
func New() (*Validator, error) {
	v := validator.New(validator.WithRequiredStructEnabled())

	// Report errors using JSON field names so the frontend can map them to inputs.
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})

	rules := map[string]validator.Func{
		"entity_type": func(fl validator.FieldLevel) bool {
			return model.EntityType(fl.Field().String()).Valid()
		},
		"entity_status": func(fl validator.FieldLevel) bool {
			return model.EntityStatus(fl.Field().String()).Valid()
		},
		"latitude": func(fl validator.FieldLevel) bool {
			return inRange(fl.Field().Float(), 90)
		},
		"longitude": func(fl validator.FieldLevel) bool {
			return inRange(fl.Field().Float(), 180)
		},
		"json_object": func(fl validator.FieldLevel) bool {
			raw := fl.Field().Bytes()
			return len(raw) == 0 || (raw[0] == '{' && json.Valid(raw))
		},
		// bcrypt only uses the first 72 bytes, so longer passwords are rejected
		// instead of silently truncated. "max" would count characters, not bytes.
		"max_bytes": func(fl validator.FieldLevel) bool {
			limit, err := strconv.Atoi(fl.Param())
			return err == nil && len(fl.Field().String()) <= limit
		},
	}
	for tag, fn := range rules {
		if err := v.RegisterValidation(tag, fn); err != nil {
			return nil, fmt.Errorf("register validation %q: %w", tag, err)
		}
	}

	return &Validator{v: v}, nil
}

// inRange reports whether f is finite and within [-limit, limit].
func inRange(f, limit float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0) && f >= -limit && f <= limit
}

// Entity normalizes in (trimming strings) and validates it.
// It returns nil when the input is valid.
func (val *Validator) Entity(in *model.EntityInput) (FieldErrors, error) {
	in.Normalize()
	return val.check(in)
}

// Location validates a location-only update. It returns nil when the input is valid.
func (val *Validator) Location(in *model.LocationInput) (FieldErrors, error) {
	return val.check(in)
}

// Register normalizes in (trimming and lowercasing the email) and validates it.
// It returns nil when the input is valid.
func (val *Validator) Register(in *model.RegisterInput) (FieldErrors, error) {
	in.Normalize()
	return val.check(in)
}

// Login normalizes in and checks that both fields are present.
// It returns nil when the input is valid.
func (val *Validator) Login(in *model.LoginInput) (FieldErrors, error) {
	in.Normalize()
	return val.check(in)
}

// Filter validates list filters. It returns nil when the filter is valid.
func (val *Validator) Filter(f *model.EntityFilter) (FieldErrors, error) {
	return val.check(f)
}

func (val *Validator) check(s any) (FieldErrors, error) {
	err := val.v.Struct(s)
	if err == nil {
		return nil, nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return nil, fmt.Errorf("validate: %w", err)
	}
	fields := make(FieldErrors, len(verrs))
	for _, fe := range verrs {
		fields[fe.Field()] = message(fe)
	}
	return fields, nil
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "max_bytes":
		return fmt.Sprintf("must be at most %s bytes", fe.Param())
	case "email":
		return "must be a valid email address"
	case "entity_type":
		return "must be one of: " + join(model.EntityTypes)
	case "entity_status":
		return "must be one of: " + join(model.EntityStatuses)
	case "latitude":
		return "must be between -90 and 90"
	case "longitude":
		return "must be between -180 and 180"
	case "json_object":
		return "must be a JSON object"
	default:
		return "is invalid"
	}
}

func join[T ~string](values []T) string {
	s := make([]string, len(values))
	for i, v := range values {
		s[i] = string(v)
	}
	return strings.Join(s, ", ")
}

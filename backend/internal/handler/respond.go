package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

const maxBodyBytes = 1 << 20 // 1 MiB

type errorBody struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Fields  validation.FieldErrors `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, map[string]any{"data": data})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: code, Message: message})
}

func writeValidation(w http.ResponseWriter, fields validation.FieldErrors) {
	writeJSON(w, http.StatusUnprocessableEntity, errorBody{Error: "validation_failed", Fields: fields})
}

// writeServiceError maps service errors to responses without leaking internals.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "entity not found")
		return
	}
	slog.Error("internal error", "error", err, "method", r.Method, "path", r.URL.Path)
	writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}

// decodeJSON decodes a single JSON object into dst, rejecting unknown fields.
// On failure it writes the error response and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var (
			typeErr *json.UnmarshalTypeError
			maxErr  *http.MaxBytesError
		)
		switch {
		case errors.Is(err, io.EOF):
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must not be empty")
		case errors.As(err, &typeErr) && typeErr.Field != "":
			// A well-formed body with a wrongly typed field is a validation error.
			writeValidation(w, validation.FieldErrors{typeErr.Field: typeErrorMessage(typeErr)})
		case errors.As(err, &maxErr):
			writeError(w, http.StatusBadRequest, "invalid_json",
				fmt.Sprintf("request body must not exceed %d bytes", maxErr.Limit))
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			writeError(w, http.StatusBadRequest, "invalid_json",
				strings.TrimPrefix(err.Error(), "json: "))
		default:
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must be a valid JSON object")
		}
		return false
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain a single JSON object")
		return false
	}
	return true
}

func typeErrorMessage(err *json.UnmarshalTypeError) string {
	kind := jsonKind(err.Type)
	// A JSON number that does not fit the Go type, e.g. 1e400 for a float64.
	if kind == "number" && strings.HasPrefix(err.Value, "number") {
		return "is out of range"
	}
	return "must be a " + kind
}

func jsonKind(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	default:
		return "valid value"
	}
}

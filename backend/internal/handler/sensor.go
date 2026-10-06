package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

const (
	defaultReadingHours = 24
	maxReadingHours     = 7 * 24
)

type sensorHandler struct {
	svc *service.SensorService
	val *validation.Validator
}

// get returns the sensor config, or {"data": null} when none was set yet.
func (h *sensorHandler) get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, cfg)
}

func (h *sensorHandler) put(w http.ResponseWriter, r *http.Request) {
	var in model.SensorInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Sensor(&in) }) {
		return
	}
	cfg, err := h.svc.PutMetric(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, cfg)
}

func (h *sensorHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// createKey returns a new device key. This is the only time it is visible.
func (h *sensorHandler) createKey(w http.ResponseWriter, r *http.Request) {
	key, err := h.svc.CreateKey(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	// The key is a secret: no browser or proxy may keep a copy.
	w.Header().Set("Cache-Control", "no-store")
	writeData(w, http.StatusCreated, key)
}

func (h *sensorHandler) readings(w http.ResponseWriter, r *http.Request) {
	hours := defaultReadingHours
	if raw := r.URL.Query().Get("hours"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxReadingHours {
			writeValidation(w, validation.FieldErrors{"hours": "must be a whole number from 1 to 168"})
			return
		}
		hours = n
	}
	data, err := h.svc.Readings(r.Context(), chi.URLParam(r, "id"), hours)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, data)
}

// deviceReading is called by the device itself, authenticated by its API key
// instead of a session. The key is checked before the body is read.
func (h *sensorHandler) deviceReading(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "id")
	spec, err := h.svc.DeviceMetric(r.Context(), entityID, bearerToken(r))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	var in model.ReadingInput
	if !decodeJSON(w, r, &in) {
		return
	}
	now := h.svc.Now()
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Reading(&in, spec, now) }) {
		return
	}

	reading := model.Reading{Value: *in.Value, RecordedAt: now}
	if in.RecordedAt != nil {
		// Validated above.
		at, _ := time.Parse(time.RFC3339, *in.RecordedAt)
		reading.RecordedAt = at.UTC().Truncate(time.Second)
	}
	if err := h.svc.Record(r.Context(), entityID, spec.ID, []model.Reading{reading}); err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusCreated, reading)
}

// bearerToken returns the token of an "Authorization: Bearer <token>" header, or "".
func bearerToken(r *http.Request) string {
	scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

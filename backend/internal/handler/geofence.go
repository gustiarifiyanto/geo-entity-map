package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

type geofenceHandler struct {
	svc *service.GeofenceService
	val *validation.Validator
}

// get returns the zone, or {"data": null} when none was set yet.
func (h *geofenceHandler) get(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, g)
}

func (h *geofenceHandler) put(w http.ResponseWriter, r *http.Request) {
	var in model.GeofenceInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Geofence(&in) }) {
		return
	}
	g, err := h.svc.Put(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, g)
}

func (h *geofenceHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *geofenceHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, list)
}

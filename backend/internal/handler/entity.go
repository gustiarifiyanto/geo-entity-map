// Package handler implements the HTTP API: decoding, validation and responses.
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

type entityHandler struct {
	svc *service.EntityService
	val *validation.Validator
}

type metaResponse struct {
	Types        []model.EntityType                      `json:"types"`
	Statuses     []model.EntityStatus                    `json:"statuses"`
	Capabilities map[model.EntityType][]model.Capability `json:"capabilities"`
}

func (h *entityHandler) meta(w http.ResponseWriter, _ *http.Request) {
	writeData(w, http.StatusOK, metaResponse{
		Types:        model.EntityTypes,
		Statuses:     model.EntityStatuses,
		Capabilities: model.TypeCapabilities,
	})
}

func (h *entityHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.EntityFilter{
		Type:   model.EntityType(q.Get("type")),
		Status: model.EntityStatus(q.Get("status")),
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Filter(&f) }) {
		return
	}

	entities, err := h.svc.List(r.Context(), f)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, entities)
}

func (h *entityHandler) get(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, e)
}

func (h *entityHandler) create(w http.ResponseWriter, r *http.Request) {
	var in model.EntityInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Entity(&in) }) {
		return
	}

	e, err := h.svc.Create(r.Context(), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/entities/"+e.ID)
	writeData(w, http.StatusCreated, e)
}

func (h *entityHandler) update(w http.ResponseWriter, r *http.Request) {
	var in model.EntityInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Entity(&in) }) {
		return
	}

	e, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, e)
}

func (h *entityHandler) updateLocation(w http.ResponseWriter, r *http.Request) {
	var in model.LocationInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Location(&in) }) {
		return
	}

	e, err := h.svc.UpdateLocation(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, e)
}

func (h *entityHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validInput runs a validation and writes a 422 (or 500) response when it fails.
func validInput(w http.ResponseWriter, r *http.Request, validate func() (validation.FieldErrors, error)) bool {
	fields, err := validate()
	if err != nil {
		writeServiceError(w, r, err)
		return false
	}
	if fields != nil {
		writeValidation(w, fields)
		return false
	}
	return true
}

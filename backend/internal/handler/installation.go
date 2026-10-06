package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

type installationHandler struct {
	svc *service.InstallationService
	val *validation.Validator
}

// get returns the schedule, or {"data": null} when none was set yet.
func (h *installationHandler) get(w http.ResponseWriter, r *http.Request) {
	inst, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, inst)
}

func (h *installationHandler) put(w http.ResponseWriter, r *http.Request) {
	var in model.InstallationInput
	if !decodeJSON(w, r, &in) {
		return
	}
	// "Today" comes from the service so it uses the app's time zone.
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Installation(&in, h.svc.Today()) }) {
		return
	}

	inst, err := h.svc.Put(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, inst)
}

func (h *installationHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *installationHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, list)
}

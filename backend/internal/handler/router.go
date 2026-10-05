package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

// NewRouter returns the HTTP handler for the whole API.
func NewRouter(svc *service.EntityService, val *validation.Validator) http.Handler {
	h := &entityHandler{svc: svc, val: val}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/meta", h.meta)
		r.Route("/entities", func(r chi.Router) {
			r.Get("/", h.list)
			r.Post("/", h.create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.get)
				r.Put("/", h.update)
				r.Delete("/", h.delete)
				r.Patch("/location", h.updateLocation)
			})
		})
	})

	return r
}

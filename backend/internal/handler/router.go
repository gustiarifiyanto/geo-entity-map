package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

// Options configures the router.
type Options struct {
	// SecureCookie marks the session cookie Secure (HTTPS only).
	SecureCookie bool
}

// NewRouter returns the HTTP handler for the whole API.
func NewRouter(
	entities *service.EntityService,
	auth *service.AuthService,
	photos *service.PhotoService,
	val *validation.Validator,
	opts Options,
) http.Handler {
	h := &entityHandler{svc: entities, val: val}
	a := &authHandler{svc: auth, val: val, secureCookie: opts.SecureCookie}
	p := &photoHandler{svc: photos}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", a.register)
			r.Post("/login", a.login)
			r.Post("/logout", a.logout)
			r.With(a.requireUser).Get("/me", a.me)
		})

		// Everything else needs a login; changing entities needs the admin role.
		// Auth runs before decoding, so an anonymous request gets 401, never 422.
		r.Group(func(r chi.Router) {
			r.Use(a.requireUser)
			r.Get("/meta", h.meta)
			r.With(requireAdmin).Get("/admin/stats", a.stats)
			r.Route("/entities", func(r chi.Router) {
				r.Get("/", h.list)
				r.With(requireAdmin).Post("/", h.create)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", h.get)
					r.With(requireAdmin).Put("/", h.update)
					r.With(requireAdmin).Delete("/", h.delete)
					r.With(requireAdmin).Patch("/location", h.updateLocation)
					r.Get("/photos", p.list)
					r.With(requireAdmin).Post("/photos", p.upload)
				})
			})
			r.Route("/photos/{photoID}", func(r chi.Router) {
				r.Get("/", p.get)
				r.With(requireAdmin).Delete("/", p.delete)
			})
		})
	})

	return r
}

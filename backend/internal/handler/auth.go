package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

const sessionCookie = "session"

type authHandler struct {
	svc          *service.AuthService
	val          *validation.Validator
	secureCookie bool
	// demo is empty unless the server runs in demo mode.
	demo []model.DemoAccount
}

type userKey struct{}

// currentUser returns the user stored by requireUser.
func currentUser(r *http.Request) (model.User, bool) {
	u, ok := r.Context().Value(userKey{}).(model.User)
	return u, ok
}

// requireUser rejects requests without a valid session with 401 and stores
// the logged-in user in the request context.
func (h *authHandler) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := h.svc.Authenticate(r.Context(), sessionToken(r))
		if errors.Is(err, model.ErrUnauthenticated) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "login required")
			return
		}
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}

// requireAdmin rejects non-admins with 403. It must run after requireUser.
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, ok := currentUser(r); !ok || u.Role != model.RoleAdmin {
			writeError(w, http.StatusForbidden, "forbidden", "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var in model.RegisterInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Register(&in) }) {
		return
	}

	s, err := h.svc.Register(r.Context(), in)
	if errors.Is(err, model.ErrEmailTaken) {
		writeValidation(w, validation.FieldErrors{"email": "is already registered"})
		return
	}
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	h.replaceSession(w, r, s)
	writeData(w, http.StatusCreated, s.User)
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var in model.LoginInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if !validInput(w, r, func() (validation.FieldErrors, error) { return h.val.Login(&in) }) {
		return
	}

	s, err := h.svc.Login(r.Context(), in)
	if errors.Is(err, model.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	h.replaceSession(w, r, s)
	writeData(w, http.StatusOK, s.User)
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if token := sessionToken(r); token != "" {
		if err := h.svc.Logout(r.Context(), token); err != nil {
			writeServiceError(w, r, err)
			return
		}
	}
	h.setCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	u, _ := currentUser(r)
	writeData(w, http.StatusOK, u)
}

// replaceSession ends the session the browser already had (if any), so
// logging in again does not leave an orphaned session, and sets the new cookie.
func (h *authHandler) replaceSession(w http.ResponseWriter, r *http.Request, s service.Session) {
	if old := sessionToken(r); old != "" {
		// Best effort: the old session expires on its own if this fails.
		_ = h.svc.Logout(r.Context(), old)
	}
	h.setCookie(w, s.Token, int(service.SessionTTL/time.Second))
}

// setCookie writes the session cookie. A negative maxAge deletes it.
func (h *authHandler) setCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func sessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

// demoAccounts lists the demo logins for the login page. It is public and
// returns an empty list unless the server runs with DEMO_ACCOUNTS=true.
func (h *authHandler) demoAccounts(w http.ResponseWriter, _ *http.Request) {
	accounts := h.demo
	if accounts == nil {
		accounts = []model.DemoAccount{}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeData(w, http.StatusOK, accounts)
}

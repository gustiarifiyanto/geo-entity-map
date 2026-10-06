package handler_test

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

func decodeUser(t *testing.T, rec *httptest.ResponseRecorder) model.User {
	t.Helper()
	return decode[struct{ Data model.User }](t, rec).Data
}

// sessionCookie returns the session cookie set by a response, failing if there is none.
func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatalf("response did not set the session cookie; headers: %v", rec.Header())
	return nil
}

func register(t *testing.T, app testApp, email, password string) *http.Cookie {
	t.Helper()
	rec := do(t, app.router, http.MethodPost, "/api/auth/register",
		`{"email": "`+email+`", "password": "`+password+`"}`)
	expectStatus(t, rec, http.StatusCreated)
	return sessionCookie(t, rec)
}

func TestRegister(t *testing.T) {
	app := newApp(t)
	rec := do(t, app.router, http.MethodPost, "/api/auth/register",
		`{"email": "  Budi@Example.COM ", "password": "rahasia123"}`)
	expectStatus(t, rec, http.StatusCreated)

	u := decodeUser(t, rec)
	if u.ID == "" || u.Email != "budi@example.com" || u.Role != model.RoleUser || u.CreatedAt.IsZero() {
		t.Errorf("unexpected user: %+v", u)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("response leaks password data: %s", rec.Body)
	}

	c := sessionCookie(t, rec)
	if c.Value == "" || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 7*24*60*60 {
		t.Errorf("unexpected session cookie: %+v", c)
	}

	// Registering logs the user in.
	rec = do(t, app.router, http.MethodGet, "/api/auth/me", "", c)
	expectStatus(t, rec, http.StatusOK)
	if got := decodeUser(t, rec); got.ID != u.ID {
		t.Errorf("me = %+v, want %+v", got, u)
	}
}

func TestRegisterErrors(t *testing.T) {
	app := newApp(t)
	register(t, app, "budi@example.com", "rahasia123")

	tests := []struct {
		name   string
		body   string
		status int
		code   string
		fields map[string]string
	}{
		{"missing fields", `{}`, http.StatusUnprocessableEntity, "validation_failed",
			map[string]string{"email": "is required", "password": "is required"}},
		{"invalid email and short password", `{"email": "budi", "password": "1234567"}`,
			http.StatusUnprocessableEntity, "validation_failed",
			map[string]string{"email": "must be a valid email address", "password": "must be at least 8 characters"}},
		{"email already registered, other case", `{"email": "BUDI@example.com", "password": "rahasia123"}`,
			http.StatusUnprocessableEntity, "validation_failed",
			map[string]string{"email": "is already registered"}},
		// Clients cannot choose their own role.
		{"role field", `{"email": "eve@example.com", "password": "rahasia123", "role": "admin"}`,
			http.StatusBadRequest, "invalid_json", nil},
		{"malformed JSON", `{"email":`, http.StatusBadRequest, "invalid_json", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, app.router, http.MethodPost, "/api/auth/register", tc.body)
			got := expectError(t, rec, tc.status, tc.code)
			if tc.fields != nil && !maps.Equal(got.Fields, tc.fields) {
				t.Errorf("fields = %v, want %v", got.Fields, tc.fields)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Errorf("failed register set cookies: %v", rec.Result().Cookies())
			}
		})
	}
}

func TestLogin(t *testing.T) {
	app := newApp(t)
	register(t, app, "budi@example.com", "rahasia123")

	rec := do(t, app.router, http.MethodPost, "/api/auth/login",
		`{"email": " BUDI@example.com", "password": "rahasia123"}`)
	expectStatus(t, rec, http.StatusOK)
	if u := decodeUser(t, rec); u.Email != "budi@example.com" || u.Role != model.RoleUser {
		t.Errorf("unexpected user: %+v", u)
	}

	rec = do(t, app.router, http.MethodGet, "/api/auth/me", "", sessionCookie(t, rec))
	expectStatus(t, rec, http.StatusOK)
}

func TestLoginErrors(t *testing.T) {
	app := newApp(t)
	register(t, app, "budi@example.com", "rahasia123")

	tests := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"wrong password", `{"email": "budi@example.com", "password": "rahasia124"}`,
			http.StatusUnauthorized, "invalid_credentials"},
		{"password with extra space", `{"email": "budi@example.com", "password": "rahasia123 "}`,
			http.StatusUnauthorized, "invalid_credentials"},
		{"unknown email", `{"email": "nobody@example.com", "password": "rahasia123"}`,
			http.StatusUnauthorized, "invalid_credentials"},
		{"short password is not a validation error", `{"email": "budi@example.com", "password": "x"}`,
			http.StatusUnauthorized, "invalid_credentials"},
		{"missing fields", `{}`, http.StatusUnprocessableEntity, "validation_failed"},
	}
	var messages []string
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, app.router, http.MethodPost, "/api/auth/login", tc.body)
			got := expectError(t, rec, tc.status, tc.code)
			if tc.code == "invalid_credentials" {
				messages = append(messages, got.Message)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Errorf("failed login set cookies: %v", rec.Result().Cookies())
			}
		})
	}
	// The response must not reveal whether the email exists.
	for _, m := range messages {
		if m != messages[0] {
			t.Errorf("invalid_credentials messages differ: %q", messages)
			break
		}
	}
}

func TestLoginReplacesPreviousSession(t *testing.T) {
	app := newApp(t)
	old := register(t, app, "budi@example.com", "rahasia123")

	rec := do(t, app.router, http.MethodPost, "/api/auth/login",
		`{"email": "budi@example.com", "password": "rahasia123"}`, old)
	expectStatus(t, rec, http.StatusOK)

	expectError(t, do(t, app.router, http.MethodGet, "/api/auth/me", "", old), http.StatusUnauthorized, "unauthorized")
	expectStatus(t, do(t, app.router, http.MethodGet, "/api/auth/me", "", sessionCookie(t, rec)), http.StatusOK)
}

func TestMeWithoutValidSession(t *testing.T) {
	app := newApp(t)
	tests := []struct {
		name    string
		cookies []*http.Cookie
	}{
		{"no cookie", nil},
		{"empty cookie", []*http.Cookie{{Name: "session", Value: ""}}},
		{"unknown token", []*http.Cookie{{Name: "session", Value: "not-a-real-token"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, app.router, http.MethodGet, "/api/auth/me", "", tc.cookies...)
			expectError(t, rec, http.StatusUnauthorized, "unauthorized")
		})
	}
}

func TestExpiredSession(t *testing.T) {
	app := newApp(t)
	c := register(t, app, "budi@example.com", "rahasia123")
	if _, err := app.db.Exec(`UPDATE sessions SET expires_at = '2000-01-01T00:00:00Z'`); err != nil {
		t.Fatalf("expire sessions: %v", err)
	}
	expectError(t, do(t, app.router, http.MethodGet, "/api/auth/me", "", c), http.StatusUnauthorized, "unauthorized")
}

func TestLogout(t *testing.T) {
	app := newApp(t)
	c := register(t, app, "budi@example.com", "rahasia123")

	rec := do(t, app.router, http.MethodPost, "/api/auth/logout", "", c)
	expectStatus(t, rec, http.StatusNoContent)
	if cleared := sessionCookie(t, rec); cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("logout did not clear the cookie: %+v", cleared)
	}

	// The old token no longer works, even if a client keeps sending it.
	expectError(t, do(t, app.router, http.MethodGet, "/api/auth/me", "", c), http.StatusUnauthorized, "unauthorized")

	// Logging out without a session is not an error.
	expectStatus(t, do(t, app.router, http.MethodPost, "/api/auth/logout", ""), http.StatusNoContent)
}

func TestAccessControl(t *testing.T) {
	app := newApp(t)
	admin := app.login(t, adminEmail, adminPassword)
	user := register(t, app, "budi@example.com", "rahasia123")

	// An entity for the routes that need an id; created by the admin.
	rec := do(t, app.router, http.MethodPost, "/api/entities", validBody, admin)
	expectStatus(t, rec, http.StatusCreated)
	id := decodeEntity(t, rec).ID

	routes := []struct {
		method, path, body string
		adminStatus        int
		read               bool // allowed for role user
	}{
		{http.MethodGet, "/api/meta", "", http.StatusOK, true},
		{http.MethodGet, "/api/entities", "", http.StatusOK, true},
		{http.MethodGet, "/api/entities/" + id, "", http.StatusOK, true},
		{http.MethodPost, "/api/entities", validBody, http.StatusCreated, false},
		{http.MethodPut, "/api/entities/" + id, validBody, http.StatusOK, false},
		{http.MethodPatch, "/api/entities/" + id + "/location", `{"latitude": 1, "longitude": 2}`, http.StatusOK, false},
		{http.MethodDelete, "/api/entities/" + id, "", http.StatusNoContent, false},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			expectError(t, do(t, app.router, rt.method, rt.path, rt.body), http.StatusUnauthorized, "unauthorized")

			rec := do(t, app.router, rt.method, rt.path, rt.body, user)
			if rt.read {
				expectStatus(t, rec, http.StatusOK)
			} else {
				expectError(t, rec, http.StatusForbidden, "forbidden")
			}

			expectStatus(t, do(t, app.router, rt.method, rt.path, rt.body, admin), rt.adminStatus)
		})
	}
}

func TestAuthRunsBeforeValidation(t *testing.T) {
	app := newApp(t)
	user := register(t, app, "budi@example.com", "rahasia123")

	// An invalid body must not reveal validation rules to someone who may not write.
	expectError(t, do(t, app.router, http.MethodPost, "/api/entities", `{}`), http.StatusUnauthorized, "unauthorized")
	expectError(t, do(t, app.router, http.MethodPost, "/api/entities", `{}`, user), http.StatusForbidden, "forbidden")
}

func TestEnsureAdmin(t *testing.T) {
	app := newApp(t)
	ctx := context.Background()

	// newApp already created the admin, so a second run changes nothing.
	u, created, err := app.auth.EnsureAdmin(ctx, model.RegisterInput{Email: adminEmail, Password: "another-password"})
	if err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	if created || u.Role != model.RoleAdmin {
		t.Errorf("EnsureAdmin on existing admin: created = %v, user = %+v", created, u)
	}
	// The original password still works.
	app.login(t, adminEmail, adminPassword)

	// A registered user is never promoted by EnsureAdmin.
	register(t, app, "budi@example.com", "rahasia123")
	u, created, err = app.auth.EnsureAdmin(ctx, model.RegisterInput{Email: "budi@example.com", Password: "rahasia123"})
	if err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	if created || u.Role != model.RoleUser {
		t.Errorf("EnsureAdmin on existing user: created = %v, user = %+v", created, u)
	}
}

package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/handler"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
)

type demoAccountsResponse struct {
	Data []model.DemoAccount `json:"data"`
}

func TestDemoAccountsOffByDefault(t *testing.T) {
	app := newApp(t)
	rec := do(t, app.router, http.MethodGet, "/api/auth/demo-accounts", "")
	expectStatus(t, rec, http.StatusOK)
	if got := rec.Body.String(); got != "{\"data\":[]}\n" {
		t.Errorf("body = %q, want an empty list", got)
	}
}

func TestDemoAccountsArePublicAndLogIn(t *testing.T) {
	app := newAppWith(t, func(auth *service.AuthService) handler.Options {
		accounts, err := auth.SeedDemoAccounts(context.Background())
		if err != nil {
			t.Fatalf("seed demo accounts: %v", err)
		}
		return handler.Options{DemoAccounts: accounts}
	})

	// No cookie: the login page needs this before anyone is logged in.
	rec := do(t, app.router, http.MethodGet, "/api/auth/demo-accounts", "")
	expectStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	accounts := decode[demoAccountsResponse](t, rec).Data
	if len(accounts) != 2 {
		t.Fatalf("got %d demo accounts, want 2: %+v", len(accounts), accounts)
	}

	for _, acc := range accounts {
		cookie := app.login(t, acc.Email, acc.Password)
		me := decodeUser(t, do(t, app.router, http.MethodGet, "/api/auth/me", "", cookie))
		if me.Email != acc.Email || me.Role != acc.Role {
			t.Errorf("logged in as %+v, want %s (%s)", me, acc.Email, acc.Role)
		}
	}
}

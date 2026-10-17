package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/auth/authtest"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/db/dbtest"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

// "provisioned" means stubbed rather than using the real test database.

func provisionedUser() generated.User {
	return generated.User{
		ID:    "user-1",
		Sub:   "sub-1",
		Name:  "Ada Lovelace",
		Email: "ada-lovelace@example.com",
	}
}

func provisionedUsers() db.Client {
	return &dbtest.Stub{User: provisionedUser()}
}

func provisionedClients() *clients.Clients {
	return &clients.Clients{DB: provisionedUsers()}
}

func newTestAPI(
	t *testing.T, idp *authtest.IDP, dbClient db.Client,
) (chi.Router, *huma.Group, *huma.Group) {
	t.Helper()

	router := chi.NewMux()
	public, secured := NewAPI(
		router,
		huma.DefaultConfig("test", "0.0.0"),
		idp.Cache,
		idp.Config(),
		dbClient,
		zap.NewNop(),
	)

	return router, public, secured
}

func testRouter(t *testing.T) *chi.Mux {
	t.Helper()

	corsHandler, err := httpx.NewCORSHandler(httpx.CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	return NewRouter(corsHandler)
}

func newHandledRouter(t *testing.T, idp *authtest.IDP) http.Handler {
	t.Helper()

	router := testRouter(t)

	if err := Handle(
		t.Context(),
		zap.NewNop(),
		router,
		provisionedClients(),
		huma.DefaultConfig("test", "1"),
		idp.Config(),
	); err != nil {
		t.Fatal(err)
	}

	return router
}

func get(router http.Handler, path, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)

	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/auth/authtest"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db/dbtest"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

func provisionedClients(t *testing.T) (*clients.Clients, *authtest.Client) {
	t.Helper()

	database, _ := dbtest.Provision(t)

	authClient := authtest.New(t, database)

	return &clients.Clients{DB: database, Auth: authClient.Client}, authClient
}

type testAPI struct {
	router  chi.Router
	public  *huma.Group
	secured *huma.Group
	client  *authtest.Client
}

func newTestAPI(t *testing.T) testAPI {
	t.Helper()

	appClients, authClient := provisionedClients(t)

	router := chi.NewMux()
	public, secured := newAPI(
		router, huma.DefaultConfig("test", "0.0.0"), appClients.Auth,
	)

	return testAPI{router: router, public: public, secured: secured, client: authClient}
}

func testRouter(t *testing.T) *chi.Mux {
	t.Helper()

	return testRouterWithLogger(t, zap.NewNop())
}

func testRouterWithLogger(t *testing.T, logger *zap.Logger) *chi.Mux {
	t.Helper()

	corsHandler, err := httpx.NewCORSHandler(httpx.CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	return NewRouter(logger, corsHandler)
}

func newHandledRouter(t *testing.T) (http.Handler, *authtest.Client) {
	t.Helper()

	router := testRouter(t)
	appClients, authClient := provisionedClients(t)

	Handle(
		zap.NewNop(),
		router,
		appClients,
		huma.DefaultConfig("test", "1"),
	)

	return router, authClient
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

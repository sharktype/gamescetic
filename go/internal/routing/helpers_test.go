package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/auth/authtest"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/db/dbtest"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

type testAPI struct {
	router  chi.Router
	public  *huma.Group
	secured *huma.Group
	signer  *authtest.Signer
}

// newTestAPI backs auth with a closed database: a request reaching the user lookup fails with 500.
func newTestAPI(t *testing.T) testAPI {
	t.Helper()

	authClient := authtest.New(t, dbtest.NewClosed(t))

	router := chi.NewMux()
	public, secured := newAPI(
		router, huma.DefaultConfig("test", "0.0.0"), authClient.Client,
	)

	return testAPI{router: router, public: public, secured: secured, signer: authClient.Signer}
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

func newHandledRouter(t *testing.T, database *db.Store) (http.Handler, *authtest.Signer) {
	t.Helper()

	router := testRouter(t)
	authClient := authtest.New(t, database)

	Handle(
		zap.NewNop(),
		router,
		&clients.Clients{DB: database, Auth: authClient.Client},
		huma.DefaultConfig("test", "1"),
	)

	return router, authClient.Signer
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

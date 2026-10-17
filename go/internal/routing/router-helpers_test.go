package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	test_db "github.com/sharktype/satchel/internal/db/test-db"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

const probePath = "/probe"

type stubInput struct{}

type stubOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

// "provisioned" means stubbed rather than using the real test database.

func provisionedUser() generated.User {
	return generated.User{
		ID:    "user-1",
		Sub:   "sub-1",
		Name:  "Ada Lovelace",
		Email: "ada-lovelace@example.com",
	}
}

func provisionedClients() *clients.Clients {
	return &clients.Clients{DB: &test_db.Stub{User: provisionedUser()}}
}

func newTestAPI(t *testing.T, idp *testIDP, appClients *clients.Clients) (chi.Router, huma.API) {
	t.Helper()

	router := chi.NewMux()
	public, secured := NewAPI(
		router,
		huma.DefaultConfig("test", "0.0.0"),
		idp.cache,
		idp.config(),
		appClients,
		zap.NewNop(),
	)

	return router, api
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

func newHandledRouter(t *testing.T, idp *testIDP) http.Handler {
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

type probe struct {
	ran     bool
	subject string
	viewer  generated.User
}

func registerProbe(api huma.API, security []map[string][]string) *probe {
	recorded := &probe{}

	huma.Register(api, huma.Operation{
		OperationID: "Probe",
		Method:      http.MethodGet,
		Path:        probePath,
		Security:    security,
	}, func(ctx context.Context, _ *stubInput) (*stubOutput, error) {
		recorded.ran = true
		recorded.subject, _ = auth.SubjectFromContext(ctx)
		recorded.viewer, _ = auth.ViewerFromContext(ctx)

		return &stubOutput{}, nil
	})

	return recorded
}

func requireJWT(scopes ...string) []map[string][]string {
	return []map[string][]string{{auth.DefaultJWTSchemeKey: scopes}}
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

package routing

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

type stubInput struct{}

type stubOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func newTestAPI(t *testing.T, idp *testIDP, appClients *clients.Clients) (chi.Router, huma.API) {
	t.Helper()

	router := chi.NewMux()
	api := MustNewAPI(
		router,
		huma.DefaultConfig("test", "0.0.0"),
		idp.cache,
		idp.config(),
		appClients,
		zap.NewNop(),
	)

	return router, api
}

func newStubAPI(t *testing.T, idp *testIDP) (chi.Router, huma.API) {
	t.Helper()

	return newTestAPI(t, idp, clients.NewStubs())
}

func testRouter(t *testing.T) *chi.Mux {
	t.Helper()

	return MustNewRouter(httpx.MustNewCORSHandler(httpx.CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
	}))
}

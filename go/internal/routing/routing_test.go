package routing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

func newHandledRouter(t *testing.T, idp *testIDP) http.Handler {
	t.Helper()

	router := MustNewRouter(httpx.MustNewCORSHandler(httpx.CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
	}))

	MustHandle(
		t.Context(),
		zap.NewNop(),
		router,
		clients.NewStubs(),
		huma.DefaultConfig("test", "1"),
		idp.config(),
	)

	return router
}

func TestMustHandle(t *testing.T) {
	t.Run("the specification documents every operation", func(t *testing.T) {
		router := newHandledRouter(t, newTestIDP(t))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		var spec struct {
			Paths map[string]json.RawMessage `json:"paths"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
			t.Fatal(err)
		}

		for _, path := range []string{"/health", "/health/db", "/health/auth", "/health/error"} {
			if _, ok := spec.Paths[path]; !ok {
				t.Fatalf("the specification is missing %q", path)
			}
		}
	})

	t.Run("each operation is limited on its own", func(t *testing.T) {
		if publicRateLimit >= globalRateLimit {
			t.Fatalf(
				"this test needs publicRateLimit (%d) below globalRateLimit (%d)",
				publicRateLimit, globalRateLimit,
			)
		}

		idp := newTestIDP(t)
		router := newHandledRouter(t, idp)

		send := func(path, token string) int {
			request := httptest.NewRequest(http.MethodGet, path, nil)

			if token != "" {
				request.Header.Set("Authorization", "Bearer "+token)
			}

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			return response.Code
		}

		for i := range publicRateLimit {
			if code := send("/health", ""); code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := send("/health", ""); code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}

		if code := send("/health/error", idp.token(t, "sub-1")); code == http.StatusTooManyRequests {
			t.Fatal("exhausting /health also exhausted /health/error")
		}
	})

	t.Run("rejected requests still count against the endpoint", func(t *testing.T) {
		if securedRateLimit >= globalRateLimit {
			t.Fatalf(
				"this test needs securedRateLimit (%d) below globalRateLimit (%d)",
				securedRateLimit, globalRateLimit,
			)
		}

		router := newHandledRouter(t, newTestIDP(t))

		send := func() int {
			request := httptest.NewRequest(http.MethodGet, "/health/auth", nil)
			request.Header.Set("Authorization", "Bearer not-a-token")

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			return response.Code
		}

		for i := range securedRateLimit {
			if code := send(); code != http.StatusUnauthorized {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusUnauthorized)
			}
		}

		if code := send(); code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}
	})
}

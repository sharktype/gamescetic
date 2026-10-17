package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/health"
	"go.uber.org/zap"
)

type stubInput struct{}

type stubOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func TestMustNewAPI(t *testing.T) {
	t.Run("an unsecured operation answers without providing a token", func(t *testing.T) {
		idp := newTestIDP(t)

		router := chi.NewMux()
		api := MustNewAPI(
			router,
			huma.DefaultConfig("test", "0.0.0"),
			idp.cache,
			idp.JWKSURL,
			idp.Issuer,
			idp.Audience,
			clients.NewStubs(),
			zap.NewNop(),
		)

		huma.Register(api, huma.Operation{
			OperationID: "Open",
			Method:      http.MethodGet,
			Path:        "/open",
		}, func(context.Context, *stubInput) (*stubOutput, error) {
			return &stubOutput{}, nil
		})

		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/open", nil))

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("a secured operation rejects a request without providing a token", func(t *testing.T) {
		idp := newTestIDP(t)

		router := chi.NewMux()
		api := MustNewAPI(
			router,
			huma.DefaultConfig("test", "0.0.0"),
			idp.cache,
			idp.JWKSURL,
			idp.Issuer,
			idp.Audience,
			clients.NewStubs(),
			zap.NewNop(),
		)

		huma.Register(api, huma.Operation{
			OperationID: "Secured",
			Method:      http.MethodGet,
			Path:        "/secured",
			Security:    []map[string][]string{{auth.DefaultJWTSchemeKey: {}}},
		}, func(context.Context, *stubInput) (*stubOutput, error) {
			return &stubOutput{}, nil
		})

		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/secured", nil))

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want %d", response.Code, http.StatusUnauthorized)
		}
	})

	t.Run("a secured operation rejects a non-bearer header", func(t *testing.T) {
		idp := newTestIDP(t)

		router := chi.NewMux()
		api := MustNewAPI(
			router,
			huma.DefaultConfig("test", "0.0.0"),
			idp.cache,
			idp.JWKSURL,
			idp.Issuer,
			idp.Audience,
			clients.NewStubs(),
			zap.NewNop(),
		)

		huma.Register(api, huma.Operation{
			OperationID: "Secured",
			Method:      http.MethodGet,
			Path:        "/secured",
			Security:    []map[string][]string{{auth.DefaultJWTSchemeKey: {}}},
		}, func(context.Context, *stubInput) (*stubOutput, error) {
			return &stubOutput{}, nil
		})

		request := httptest.NewRequest(http.MethodGet, "/secured", nil)
		request.Header.Set("Authorization", "Basic abc")

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want %d", response.Code, http.StatusUnauthorized)
		}
	})

	t.Run("auth health answers with the viewer", func(t *testing.T) {
		idp := newTestIDP(t)

		client := db.NewStub()
		client.Row = []any{
			"user-1", "sub-1", "Ada Lovelace", "ada-lovelace@example.com", nil, time.Now(), nil,
		}

		router := chi.NewMux()
		api := MustNewAPI(
			router,
			huma.DefaultConfig("test", "0.0.0"),
			idp.cache,
			idp.JWKSURL,
			idp.Issuer,
			idp.Audience,
			&clients.Clients{DB: client},
			zap.NewNop(),
		)

		huma.Register(api, huma.Operation{
			OperationID: "GetAuthHealth",
			Method:      http.MethodGet,
			Path:        "/health/auth",
			Security:    []map[string][]string{{auth.DefaultJWTSchemeKey: {}}},
		}, health.GetAuthHealth)

		request := httptest.NewRequest(http.MethodGet, "/health/auth", nil)
		request.Header.Set("Authorization", "Bearer "+idp.token(t, "sub-1"))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		var body struct {
			Message string `json:"message"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}

		if body.Message != "user-1 OK" {
			t.Fatalf("got %q, want %q", body.Message, "user-1 OK")
		}
	})
}

func TestMustNewRouter(t *testing.T) {
	router := MustNewRouter([]string{"http://example.com"}, nil, nil, nil, false, 0)

	router.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("arbitrary panic")
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

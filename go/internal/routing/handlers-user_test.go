package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

func TestRequireUserMiddleware(t *testing.T) {
	t.Run("a user that is not provisioned is forbidden", func(t *testing.T) {
		idp := newTestIDP(t)

		client := db.NewStub()
		client.QueryRowErr = pgx.ErrNoRows

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

		handlerRan := false

		huma.Register(api, huma.Operation{
			OperationID: "Secured",
			Method:      http.MethodGet,
			Path:        "/secured",
			Security:    []map[string][]string{{auth.DefaultJWTSchemeKey: {}}},
		}, func(context.Context, *stubInput) (*stubOutput, error) {
			handlerRan = true

			return &stubOutput{}, nil
		})

		request := httptest.NewRequest(http.MethodGet, "/secured", nil)
		request.Header.Set("Authorization", "Bearer "+idp.token(t, "user-1"))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusForbidden {
			t.Fatalf("got %d, want %d", response.Code, http.StatusForbidden)
		}

		if handlerRan {
			t.Fatal("the handler ran")
		}
	})

	t.Run("a provisioned user is in the context", func(t *testing.T) {
		idp := newTestIDP(t)

		client := db.NewStub()
		client.Row = []any{
			"user-1",
			"sub-1",
			"Ada Lovelace",
			"ada-lovelace@example.com",
			nil,
			time.Now(),
			nil,
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

		var viewerID string
		var viewerEmail string

		huma.Register(api, huma.Operation{
			OperationID: "Secured",
			Method:      http.MethodGet,
			Path:        "/secured",
			Security:    []map[string][]string{{auth.DefaultJWTSchemeKey: {}}},
		}, func(ctx context.Context, _ *stubInput) (*stubOutput, error) {
			if viewer, ok := auth.ViewerFromContext(ctx); ok {
				viewerID = viewer.ID
				viewerEmail = viewer.Email
			}

			return &stubOutput{}, nil
		})

		request := httptest.NewRequest(http.MethodGet, "/secured", nil)
		request.Header.Set("Authorization", "Bearer "+idp.token(t, "sub-1"))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		if viewerID != "user-1" {
			t.Fatalf("got id %q, want %q", viewerID, "user-1")
		}

		if viewerEmail != "ada-lovelace@example.com" {
			t.Fatalf("got email %q, want %q", viewerEmail, "ada-lovelace@example.com")
		}
	})
}

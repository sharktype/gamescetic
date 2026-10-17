package routing

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db"
)

func TestNewJWTAuthMiddleware(t *testing.T) {
	t.Run("an unsecured operation never sets a subject", func(t *testing.T) {
		idp := newTestIDP(t)

		router, api := newStubAPI(t, idp)

		var subject string

		huma.Register(api, huma.Operation{
			OperationID: "Open",
			Method:      http.MethodGet,
			Path:        "/open",
		}, func(ctx context.Context, _ *stubInput) (*stubOutput, error) {
			subject, _ = auth.SubjectFromContext(ctx)

			return &stubOutput{}, nil
		})

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
		claims := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"attacker"}`))

		request := httptest.NewRequest(http.MethodGet, "/open", nil)
		request.Header.Set("Authorization", "Bearer "+header+"."+claims+".no-signature")

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		if subject != "" {
			t.Fatalf("got subject %q, want none", subject)
		}
	})

	t.Run("a verified token sets the subject", func(t *testing.T) {
		idp := newTestIDP(t)

		router, api := newStubAPI(t, idp)

		var subject string

		huma.Register(api, huma.Operation{
			OperationID: "Secured",
			Method:      http.MethodGet,
			Path:        "/secured",
			Security:    []map[string][]string{{auth.DefaultJWTSchemeKey: {}}},
		}, func(ctx context.Context, _ *stubInput) (*stubOutput, error) {
			subject, _ = auth.SubjectFromContext(ctx)

			return &stubOutput{}, nil
		})

		request := httptest.NewRequest(http.MethodGet, "/secured", nil)
		request.Header.Set("Authorization", "Bearer "+idp.token(t, "user-1"))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		if subject != "user-1" {
			t.Fatalf("got subject %q, want %q", subject, "user-1")
		}
	})

	t.Run("a token signed by another key is rejected", func(t *testing.T) {
		idp := newTestIDP(t)
		otherIDP := newTestIDP(t)

		router, api := newStubAPI(t, idp)

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
		request.Header.Set("Authorization", "Bearer "+otherIDP.token(t, "attacker"))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want %d", response.Code, http.StatusUnauthorized)
		}

		if handlerRan {
			t.Fatal("the handler ran")
		}
	})

	t.Run("a token without a subject is rejected", func(t *testing.T) {
		idp := newTestIDP(t)

		router, api := newStubAPI(t, idp)

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
		request.Header.Set("Authorization", "Bearer "+idp.token(t, ""))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want %d", response.Code, http.StatusUnauthorized)
		}

		if handlerRan {
			t.Fatal("the handler ran")
		}
	})

	t.Run("scopes", func(t *testing.T) {
		idp := newTestIDP(t)

		// "posts" is just an example here.

		tests := []struct {
			name           string
			declared       []string
			granted        []string
			expectedStatus int
			expectedIsRun  bool
		}{
			{
				name:           "no scopes declared",
				expectedStatus: http.StatusOK,
				expectedIsRun:  true,
			},
			{
				name:           "all declared scopes granted",
				declared:       []string{"posts:read", "posts:write"},
				granted:        []string{"posts:read", "posts:write"},
				expectedStatus: http.StatusOK,
				expectedIsRun:  true,
			},
			{
				name:           "extra scopes are fine",
				declared:       []string{"posts:read"},
				granted:        []string{"posts:read", "admin"},
				expectedStatus: http.StatusOK,
				expectedIsRun:  true,
			},
			{
				name:           "one of two declared scopes is not enough",
				declared:       []string{"posts:read", "posts:write"},
				granted:        []string{"posts:read"},
				expectedStatus: http.StatusForbidden,
			},
			{
				name:           "no scopes claim",
				declared:       []string{"posts:read"},
				expectedStatus: http.StatusForbidden,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
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

				router, api := newTestAPI(t, idp, &clients.Clients{DB: client})

				handlerRan := false

				huma.Register(api, huma.Operation{
					OperationID: "Secured",
					Method:      http.MethodGet,
					Path:        "/secured",
					Security:    []map[string][]string{{auth.DefaultJWTSchemeKey: test.declared}},
				}, func(context.Context, *stubInput) (*stubOutput, error) {
					handlerRan = true

					return &stubOutput{}, nil
				})

				request := httptest.NewRequest(http.MethodGet, "/secured", nil)
				request.Header.Set("Authorization", "Bearer "+idp.tokenWithScopes(t, "sub-1", test.granted))

				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)

				if response.Code != test.expectedStatus {
					t.Fatalf("got %d, want %d", response.Code, test.expectedStatus)
				}

				if handlerRan != test.expectedIsRun {
					t.Fatalf("handler ran: got %t, want %t", handlerRan, test.expectedIsRun)
				}
			})
		}
	})
}

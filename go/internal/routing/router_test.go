package routing

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/auth/authtest"
	"github.com/sharktype/satchel/internal/db/dbtest"
)

func TestNewAPI(t *testing.T) {

	t.Run("an unsecured operation never validates a token", func(t *testing.T) {
		api := newTestAPI(t)
		handler := authtest.RegisterProbe(api.public, nil)

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
		claims := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"sub-1"}`))

		forged := header + "." + claims + ".no-signature"

		code := get(api.router, authtest.ProbePath, "Bearer "+forged).Code
		if code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if handler.User != (auth.User{}) {
			t.Fatalf("got user %+v, want none", handler.User)
		}
	})

	t.Run("the public group refuses an operation declaring a JWT requirement", func(t *testing.T) {
		api := newTestAPI(t)

		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic")
			}
		}()

		authtest.RegisterProbe(api.public, authtest.RequireJWT())
	})

	t.Run("the JWT scheme is merged into caller-provided schemes", func(t *testing.T) {
		config := huma.DefaultConfig("test", "0.0.0")
		config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
			"ApiKey": {Type: "apiKey", In: "header", Name: "X-API-Key"},
		}

		authClient := authtest.New(t, dbtest.NewClosed(t))
		_, secured := newAPI(chi.NewMux(), config, authClient.Client)

		schemes := secured.OpenAPI().Components.SecuritySchemes
		if _, ok := schemes["ApiKey"]; !ok {
			t.Fatal("newAPI dropped a caller-provided security scheme")
		}

		if _, ok := schemes[auth.JWTSchemeKey]; !ok {
			t.Fatal("newAPI did not add the JWT security scheme")
		}
	})

	t.Run("the secured group refuses an operation declaring two JWT requirements", func(t *testing.T) {
		api := newTestAPI(t)

		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic")
			}
		}()

		security := append(authtest.RequireJWT(), authtest.RequireJWT()...)
		authtest.RegisterProbe(api.secured, security)
	})

	t.Run("the secured group keeps scopes and documents one JWT requirement", func(t *testing.T) {
		api := newTestAPI(t)
		handler := authtest.RegisterProbe(api.secured, authtest.RequireJWT("posts:read"))

		token := api.signer.Token(t, dbtest.ProvisionedUser().Sub)

		code := get(api.router, authtest.ProbePath, "Bearer "+token).Code
		if code != http.StatusForbidden {
			t.Fatalf("got %d, want %d", code, http.StatusForbidden)
		}

		if handler.Ran {
			t.Fatal("the handler ran")
		}

		security := api.secured.OpenAPI().Paths[authtest.ProbePath].Get.Security
		if len(security) != 1 {
			t.Fatalf("got %d documented security requirements, want 1", len(security))
		}

		if _, ok := api.secured.OpenAPI().Components.SecuritySchemes[auth.JWTSchemeKey]; !ok {
			t.Fatal("the API does not document the JWT security scheme")
		}
	})

	rejections := []struct {
		name          string
		authorization string
	}{
		{
			name: "a secured operation rejects a request without providing a token",
		},
		{
			name:          "a secured operation rejects a non-bearer header",
			authorization: "Basic abc",
		},
	}

	for _, test := range rejections {
		t.Run(test.name, func(t *testing.T) {
			api := newTestAPI(t)
			handler := authtest.RegisterProbe(api.secured, nil)

			if code := get(
				api.router, authtest.ProbePath, test.authorization,
			).Code; code != http.StatusUnauthorized {
				t.Fatalf("got %d, want %d", code, http.StatusUnauthorized)
			}

			if handler.Ran {
				t.Fatal("the handler ran")
			}
		})
	}
}

func TestDeclareJWT(t *testing.T) {
	t.Run("adds a JWT requirement when none is declared", func(t *testing.T) {
		operation := &huma.Operation{OperationID: "Test"}

		declareJWT(operation)

		if len(operation.Security) != 1 {
			t.Fatalf("got %d security requirements, want 1", len(operation.Security))
		}

		if _, ok := operation.Security[0][auth.JWTSchemeKey]; !ok {
			t.Fatal("declareJWT did not add the JWT requirement")
		}
	})

	rejections := []struct {
		name     string
		security []map[string][]string
	}{
		{
			name:     "a non-JWT requirement",
			security: []map[string][]string{{"ApiKey": {}}},
		},
		{
			name:     "an empty requirement",
			security: []map[string][]string{{}},
		},
		{
			name:     "a requirement combining JWT with another scheme",
			security: []map[string][]string{{auth.JWTSchemeKey: {}, "ApiKey": {}}},
		},
		{
			name:     "two JWT requirements",
			security: append(authtest.RequireJWT(), authtest.RequireJWT()...),
		},
	}

	for _, test := range rejections {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected a panic")
				}
			}()

			declareJWT(&huma.Operation{OperationID: "Test", Security: test.security})
		})
	}
}

func TestNewRouter(t *testing.T) {
	t.Run("panics are recovered", func(t *testing.T) {
		router := testRouter(t)

		router.Get("/panic", func(http.ResponseWriter, *http.Request) {
			panic("arbitrary panic")
		})

		if code := get(router, "/panic", "").Code; code != http.StatusInternalServerError {
			t.Fatalf("got %d, want %d", code, http.StatusInternalServerError)
		}
	})

	t.Run("the client address replaces the peer address", func(t *testing.T) {
		router := testRouter(t)

		var remoteAddr string
		router.Get("/ip", func(_ http.ResponseWriter, r *http.Request) {
			remoteAddr = r.RemoteAddr
		})

		getFrom(router, "/ip", "1.0.0.1")

		if remoteAddr != "1.0.0.1" {
			t.Fatalf("got %q, want %q", remoteAddr, "1.0.0.1")
		}
	})

	t.Run("the global limiter covers ping", func(t *testing.T) {
		router := testRouter(t)

		// chi only runs its middleware once the mux has a route.

		router.Get("/noop", func(http.ResponseWriter, *http.Request) {})

		for i := range globalRateLimit {
			if code := get(router, "/ping", "").Code; code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := get(router, "/ping", "").Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}
	})

	t.Run("the global limiter is per client behind the proxy", func(t *testing.T) {
		router := testRouter(t)

		router.Get("/noop", func(http.ResponseWriter, *http.Request) {})

		for i := range globalRateLimit {
			if code := getFrom(router, "/ping", "1.0.0.1").Code; code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := getFrom(router, "/ping", "1.0.0.1").Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}

		if code := getFrom(router, "/ping", "9.9.9.9, 1.0.0.1").Code; code != http.StatusTooManyRequests {
			t.Fatalf("a forged leftmost hop escaped the limit: got %d, want %d", code, http.StatusTooManyRequests)
		}

		if code := getFrom(router, "/ping", "1.0.0.2").Code; code != http.StatusOK {
			t.Fatalf("a second client was throttled: got %d, want %d", code, http.StatusOK)
		}
	})
}

type thingInput struct {
	ID string `path:"id"`
}

func TestRateLimit(t *testing.T) {
	t.Run("limits by operation rather than URL path", func(t *testing.T) {
		api := newTestAPI(t)

		huma.Register(api.public, huma.Operation{
			OperationID: "GetThing",
			Method:      http.MethodGet,
			Path:        "/things/{id}",
		}, func(context.Context, *thingInput) (*struct{}, error) {
			return nil, nil
		})

		for i := range publicRateLimit {
			code := get(api.router, "/things/"+strconv.Itoa(i), "").Code
			if code == http.StatusTooManyRequests {
				t.Fatalf("request %d was throttled", i+1)
			}
		}

		if code := get(api.router, "/things/other", "").Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}
	})

	t.Run("hitting the limit on one operation does not block another", func(t *testing.T) {
		api := newTestAPI(t)

		authtest.RegisterProbe(api.public, nil)

		huma.Register(api.public, huma.Operation{
			OperationID: "GetThing",
			Method:      http.MethodGet,
			Path:        "/things/{id}",
		}, func(context.Context, *thingInput) (*struct{}, error) {
			return nil, nil
		})

		for range publicRateLimit {
			get(api.router, authtest.ProbePath, "")
		}

		if code := get(api.router, "/things/a", "").Code; code == http.StatusTooManyRequests {
			t.Fatal("exhausting one operation also exhausted another in the same group")
		}
	})

	t.Run("the operation limiter is per client behind the proxy", func(t *testing.T) {
		api := newTestAPI(t)

		authtest.RegisterProbe(api.public, nil)

		for i := range publicRateLimit {
			if code := getFrom(api.router, authtest.ProbePath, "1.0.0.1").Code; code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := getFrom(api.router, authtest.ProbePath, "9.9.9.9, 1.0.0.1").Code; code != http.StatusTooManyRequests {
			t.Fatalf("a forged leftmost hop escaped the limit: got %d, want %d", code, http.StatusTooManyRequests)
		}

		if code := getFrom(api.router, authtest.ProbePath, "1.0.0.2").Code; code != http.StatusOK {
			t.Fatalf("a second client was throttled: got %d, want %d", code, http.StatusOK)
		}
	})
}

package routing

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/gamescetic/internal/auth"
	"github.com/sharktype/gamescetic/internal/auth/authtest"
	"github.com/sharktype/gamescetic/internal/db/dbtest"
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

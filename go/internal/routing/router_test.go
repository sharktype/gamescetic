package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/auth/authtest"
)

func TestNewAPI(t *testing.T) {
	idp := authtest.NewIDP(t)

	t.Run("an unsecured operation answers without providing a token", func(t *testing.T) {
		router, public, _ := newTestAPI(t, idp, provisionedUsers())
		handler := authtest.RegisterProbe(public, nil)

		if code := get(router, authtest.ProbePath, "").Code; code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if !handler.Ran {
			t.Fatal("the handler did not run")
		}
	})

	t.Run("the public group refuses an operation declaring a JWT requirement", func(t *testing.T) {
		_, public, _ := newTestAPI(t, idp, provisionedUsers())

		defer func() {
			if recover() == nil {
				t.Fatal("expected a panic")
			}
		}()

		authtest.RegisterProbe(public, authtest.RequireJWT())
	})

	t.Run("the secured group keeps scopes and documents one JWT requirement", func(t *testing.T) {
		router, _, secured := newTestAPI(t, idp, provisionedUsers())
		handler := authtest.RegisterProbe(secured, authtest.RequireJWT("posts:read"))

		code := get(router, authtest.ProbePath, "Bearer "+idp.Token(t, "sub-1")).Code
		if code != http.StatusForbidden {
			t.Fatalf("got %d, want %d", code, http.StatusForbidden)
		}

		if handler.Ran {
			t.Fatal("the handler ran")
		}

		security := secured.OpenAPI().Paths[authtest.ProbePath].Get.Security
		if len(security) != 1 {
			t.Fatalf("got %d documented security requirements, want 1", len(security))
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
			router, _, secured := newTestAPI(t, idp, provisionedUsers())
			handler := authtest.RegisterProbe(secured, nil)

			if code := get(
				router, authtest.ProbePath, test.authorization,
			).Code; code != http.StatusUnauthorized {
				t.Fatalf("got %d, want %d", code, http.StatusUnauthorized)
			}

			if handler.Ran {
				t.Fatal("the handler ran")
			}
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

		request := func(forwardedFor string) int {
			request := httptest.NewRequest(http.MethodGet, "/ping", nil)
			request.Header.Set("X-Forwarded-For", forwardedFor)

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			return response.Code
		}

		for i := range globalRateLimit {
			if code := request("1.0.0.1"); code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := request("1.0.0.1"); code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}

		if code := request("1.0.0.2"); code != http.StatusOK {
			t.Fatalf("a second client was throttled: got %d, want %d", code, http.StatusOK)
		}
	})
}

type thingInput struct {
	ID string `path:"id"`
}

func TestRateLimit(t *testing.T) {
	t.Run("limits by operation rather than URL path", func(t *testing.T) {
		router, public, _ := newTestAPI(t, authtest.NewIDP(t), provisionedUsers())

		huma.Register(public, huma.Operation{
			OperationID: "GetThing",
			Method:      http.MethodGet,
			Path:        "/things/{id}",
		}, func(context.Context, *thingInput) (*struct{}, error) {
			return nil, nil
		})

		for i := range publicRateLimit {
			code := get(router, "/things/"+strconv.Itoa(i), "").Code
			if code == http.StatusTooManyRequests {
				t.Fatalf("request %d was throttled", i+1)
			}
		}

		if code := get(router, "/things/other", "").Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}
	})

	t.Run("hitting the limit on one operation does not block another", func(t *testing.T) {
		router, public, _ := newTestAPI(t, authtest.NewIDP(t), provisionedUsers())

		authtest.RegisterProbe(public, nil)

		huma.Register(public, huma.Operation{
			OperationID: "GetThing",
			Method:      http.MethodGet,
			Path:        "/things/{id}",
		}, func(context.Context, *thingInput) (*struct{}, error) {
			return nil, nil
		})

		for range publicRateLimit {
			get(router, authtest.ProbePath, "")
		}

		if code := get(router, "/things/a", "").Code; code == http.StatusTooManyRequests {
			t.Fatal("exhausting one operation also exhausted another in the same group")
		}
	})
}

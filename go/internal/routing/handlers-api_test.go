package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

func TestMustNewAPI(t *testing.T) {
	idp := newTestIDP(t)

	t.Run("an unsecured operation answers without providing a token", func(t *testing.T) {
		router, api := newTestAPI(t, idp, provisionedClients())
		handler := registerProbe(api, nil)

		if code := get(router, probePath, "").Code; code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if !handler.ran {
			t.Fatal("the handler did not run")
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
			router, api := newTestAPI(t, idp, provisionedClients())
			handler := registerProbe(api, requireJWT())

			if code := get(
				router, probePath, test.authorization,
			).Code; code != http.StatusUnauthorized {
				t.Fatalf("got %d, want %d", code, http.StatusUnauthorized)
			}

			if handler.ran {
				t.Fatal("the handler ran")
			}
		})
	}
}

func TestMustNewRouter(t *testing.T) {
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

func TestLimitByRoute(t *testing.T) {
	t.Run("limits by route pattern rather than URL path", func(t *testing.T) {
		router := chi.NewMux()

		router.Group(func(r chi.Router) {
			r.Use(limitByRoute(map[string]func(http.Handler) http.Handler{
				"/things/{id}": httprate.Limit(2, rateLimitWindow),
			}))

			r.Get("/things/{id}", func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)
			})

			r.Get("/other", func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusOK)
			})
		})

		for _, id := range []string{"a", "b"} {
			if code := get(router, "/things/"+id, "").Code; code != http.StatusOK {
				t.Fatalf("%s: got %d, want %d", id, code, http.StatusOK)
			}
		}

		if code := get(router, "/things/c", "").Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}

		// A route that is not listed falls through to whatever the router already applied.

		if code := get(router, "/other", "").Code; code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}
	})
}

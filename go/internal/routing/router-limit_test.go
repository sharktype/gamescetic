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

type thingInput struct {
	ID string `path:"id"`
}

func TestRateLimit(t *testing.T) {
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

	t.Run("an unparseable client address still shares a bucket", func(t *testing.T) {
		api := newTestAPI(t)

		authtest.RegisterProbe(api.public, nil)

		request := func(i int) int {
			request := httptest.NewRequest(http.MethodGet, authtest.ProbePath, nil)
			request.RemoteAddr = "not-an-address-" + strconv.Itoa(i)

			response := httptest.NewRecorder()
			api.router.ServeHTTP(response, request)

			return response.Code
		}

		for i := range publicRateLimit {
			if code := request(i); code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := request(publicRateLimit); code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}
	})
}

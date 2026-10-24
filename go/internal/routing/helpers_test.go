package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/auth/authtest"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/db/dbtest"
	"github.com/sharktype/satchel/internal/httpx"
	appLogger "github.com/sharktype/satchel/internal/logger"
	"go.uber.org/zap"
)

type testAPI struct {
	router  chi.Router
	public  *huma.Group
	secured *huma.Group
	signer  *authtest.Signer
}

// newTestAPI creates a test API.
//
// The test API performs auth ops with a closed DB, meaning anything requiring DB access will fail.
func newTestAPI(t *testing.T) testAPI {
	t.Helper()

	authClient := authtest.New(t, dbtest.NewClosed(t))

	router := testRouter(t)
	public, secured := newAPI(
		router, huma.DefaultConfig("test", "0.0.0"), authClient.Client,
	)

	return testAPI{router: router, public: public, secured: secured, signer: authClient.Signer}
}

func testRouter(t *testing.T) *chi.Mux {
	t.Helper()

	return testRouterWithLogger(t, zap.NewNop())
}

func testRouterWithLogger(t *testing.T, logger *zap.Logger) *chi.Mux {
	t.Helper()

	useLogger(t, logger)

	corsHandler, err := httpx.NewCORSHandler(httpx.CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}

	return NewRouter(corsHandler)
}

func useLogger(t *testing.T, logger *zap.Logger) {
	t.Helper()

	previous := appLogger.Global()
	appLogger.Set(logger)

	t.Cleanup(func() { appLogger.Set(previous) })
}

func newHandledRouter(t *testing.T, database *db.Store) (http.Handler, *authtest.Signer) {
	t.Helper()

	router := testRouter(t)
	authClient := authtest.New(t, database)

	Handle(
		router,
		&clients.Clients{DB: database, Auth: authClient.Client},
		huma.DefaultConfig("test", "1"),
	)

	return router, authClient.Signer
}

func get(router http.Handler, path, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)

	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	return serve(router, request)
}

func getFrom(router http.Handler, path, forwardedFor string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Forwarded-For", forwardedFor)

	return serve(router, request)
}

func serve(router http.Handler, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

// assertLimitedPerClient asserts rate limits are enforced in tests.
//
// It exhausts one client's limit on path, then checks that the bucket follows the trusted client
// address rather than anything the client can forge.
func assertLimitedPerClient(t *testing.T, router http.Handler, path string, limit int) {
	t.Helper()

	clients := []struct {
		exhausted  string
		sameBucket string
		other      string
	}{
		{exhausted: "1.0.0.1", sameBucket: "1.0.0.1", other: "1.0.0.2"},
		{exhausted: "2001:db8::1", sameBucket: "2001:db8::2", other: "2001:db8:0:1::1"},
	}

	for _, client := range clients {
		for i := range limit {
			if code := getFrom(
				router, path, client.exhausted,
			).Code; code != http.StatusOK {
				t.Fatalf(
					"%s request %d: got %d, want %d", client.exhausted, i+1, code, http.StatusOK,
				)
			}
		}

		if code := getFrom(
			router, path, client.sameBucket,
		).Code; code != http.StatusTooManyRequests {
			t.Fatalf(
				"%s escaped the limit of %s: got %d, want %d",
				client.sameBucket, client.exhausted, code, http.StatusTooManyRequests,
			)
		}

		if code := getFrom(
			router, path, "9.9.9.9, "+client.exhausted,
		).Code; code != http.StatusTooManyRequests {
			t.Fatalf(
				"a forged leftmost hop escaped the limit of %s: got %d, want %d",
				client.exhausted, code, http.StatusTooManyRequests,
			)
		}

		if code := getFrom(router, path, client.other).Code; code != http.StatusOK {
			t.Fatalf(
				"%s was throttled by %s: got %d, want %d",
				client.other, client.exhausted, code, http.StatusOK,
			)
		}
	}
}

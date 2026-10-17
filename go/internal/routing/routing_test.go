package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharktype/satchel/internal/clients"
	"go.uber.org/zap"
)

func TestMustHandle(t *testing.T) {
	t.Run("docs require a token", func(t *testing.T) {
		idp := newTestIDP(t)

		router := MustNewRouter([]string{"http://example.com"}, nil, nil, nil, false, 0)

		MustHandle(
			t.Context(),
			zap.NewNop(),
			router,
			clients.NewStubs(),
			"test",
			"1",
			idp.JWKSURL,
			idp.Issuer,
			idp.Audience,
		)

		for _, path := range []string{"/docs", "/openapi.json"} {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s: got %d, want %d", path, response.Code, http.StatusUnauthorized)
			}
		}

		request := httptest.NewRequest(http.MethodGet, "/docs", nil)
		request.Header.Set("Authorization", "Bearer "+idp.token(t, "user-1"))

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}
	})
}

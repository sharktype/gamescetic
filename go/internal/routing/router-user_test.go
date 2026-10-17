package routing

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/internal/clients"
	test_db "github.com/sharktype/satchel/internal/db/test-db"
)

func TestRequireUserMiddleware(t *testing.T) {
	idp := newTestIDP(t)

	t.Run("a user that is not provisioned is forbidden", func(t *testing.T) {
		unprovisioned := &clients.Clients{DB: &test_db.Stub{UserErr: pgx.ErrNoRows}}

		router, api := newTestAPI(t, idp, unprovisioned)
		handler := registerProbe(api, requireJWT())

		if code := get(
			router, probePath, "Bearer "+idp.token(t, "user-1"),
		).Code; code != http.StatusForbidden {
			t.Fatalf("got %d, want %d", code, http.StatusForbidden)
		}

		if handler.ran {
			t.Fatal("the handler ran")
		}
	})

	t.Run("a provisioned user is in the context", func(t *testing.T) {
		router, api := newTestAPI(t, idp, provisionedClients())
		handler := registerProbe(api, requireJWT())

		if code := get(
			router, probePath, "Bearer "+idp.token(t, "sub-1"),
		).Code; code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if handler.viewer != provisionedUser() {
			t.Fatalf("got viewer %+v, want %+v", handler.viewer, provisionedUser())
		}
	})
}

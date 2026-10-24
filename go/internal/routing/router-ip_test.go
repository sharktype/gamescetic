package routing

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
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
}

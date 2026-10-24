package routing

import (
	"net/http"
	"testing"
)

func TestRecoverer(t *testing.T) {
	t.Run("panics are recovered", func(t *testing.T) {
		router := testRouter(t)

		router.Get("/panic", func(http.ResponseWriter, *http.Request) {
			panic("arbitrary panic")
		})

		if code := get(router, "/panic", "").Code; code != http.StatusInternalServerError {
			t.Fatalf("got %d, want %d", code, http.StatusInternalServerError)
		}
	})
}

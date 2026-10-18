package routing

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHandle(t *testing.T) {
	t.Run("the specification documents every operation", func(t *testing.T) {
		router, _ := newHandledRouter(t)
		response := get(router, "/openapi.json", "")

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		var spec struct {
			Paths map[string]json.RawMessage `json:"paths"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
			t.Fatal(err)
		}

		for _, path := range []string{"/health", "/health/db", "/health/auth", "/health/error"} {
			if _, ok := spec.Paths[path]; !ok {
				t.Fatalf("the specification is missing %q", path)
			}
		}
	})

	t.Run("auth health answers with the user", func(t *testing.T) {
		router, authClient := newHandledRouter(t)

		response := get(router, "/health/auth", "Bearer "+authClient.Token(t, "sub-1"))

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		var body struct {
			Message string `json:"message"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}

		if body.Message != "user-1 OK" {
			t.Fatalf("got %q, want %q", body.Message, "user-1 OK")
		}
	})

	t.Run("a handler returning no output still reports its status", func(t *testing.T) {
		router, authClient := newHandledRouter(t)

		code := get(router, "/health/error", "Bearer "+authClient.Token(t, "sub-1")).Code
		if code != http.StatusInternalServerError {
			t.Fatalf("got %d, want %d", code, http.StatusInternalServerError)
		}
	})

	t.Run("each operation is limited on its own", func(t *testing.T) {
		if publicRateLimit >= globalRateLimit {
			t.Fatalf(
				"this test needs publicRateLimit (%d) below globalRateLimit (%d)",
				publicRateLimit, globalRateLimit,
			)
		}

		router, authClient := newHandledRouter(t)

		for i := range publicRateLimit {
			if code := get(router, "/health", "").Code; code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := get(router, "/health", "").Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}

		code := get(router, "/health/error", "Bearer "+authClient.Token(t, "sub-1")).Code
		if code == http.StatusTooManyRequests {
			t.Fatal("exhausting /health also exhausted /health/error")
		}
	})

	t.Run("rejected requests still count against the endpoint", func(t *testing.T) {
		if securedRateLimit >= globalRateLimit {
			t.Fatalf(
				"this test needs securedRateLimit (%d) below globalRateLimit (%d)",
				securedRateLimit, globalRateLimit,
			)
		}

		router, _ := newHandledRouter(t)

		for i := range securedRateLimit {
			if code := get(
				router, "/health/auth", "Bearer not-a-token",
			).Code; code != http.StatusUnauthorized {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusUnauthorized)
			}
		}

		if code := get(
			router, "/health/auth", "Bearer not-a-token",
		).Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}
	})
}

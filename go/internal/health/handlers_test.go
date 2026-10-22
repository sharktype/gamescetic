package health

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	dbtest "github.com/sharktype/satchel/internal/db/dbtest"
)

func TestGetDatabaseHealth(t *testing.T) {
	t.Run("a reachable database reports OK", func(t *testing.T) {
		output, err := GetDatabaseHealth(dbtest.New(t))(t.Context(), &GetHealthInput{})
		if err != nil {
			t.Fatal(err)
		}

		if output.Body.Message != "database connection OK" {
			t.Fatalf("got %q, want %q", output.Body.Message, "database connection OK")
		}
	})

	t.Run("a failing check returns a fixed 503 that keeps the cause", func(t *testing.T) {
		_, err := GetDatabaseHealth(dbtest.Unavailable(t))(t.Context(), &GetHealthInput{})
		if err == nil {
			t.Fatal("expected an error")
		}

		var statusErr huma.StatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("got %v, want a huma.StatusError", err)
		}

		if statusErr.GetStatus() != http.StatusServiceUnavailable {
			t.Fatalf("got status %d, want %d", statusErr.GetStatus(), http.StatusServiceUnavailable)
		}

		cause := errors.Unwrap(err)
		if cause == nil {
			t.Fatal("the error lost its cause")
		}

		if strings.Contains(statusErr.Error(), cause.Error()) {
			t.Fatalf("the response error leaks the cause: %q", statusErr.Error())
		}
	})
}

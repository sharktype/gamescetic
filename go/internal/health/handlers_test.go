package health

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	dbtest "github.com/sharktype/satchel/internal/db/dbtest"
)

func TestGetDatabaseHealth(t *testing.T) {
	t.Run("a reachable database reports OK", func(t *testing.T) {
		output, err := GetDatabaseHealth(&dbtest.Stub{})(
			context.Background(), &GetHealthInput{},
		)
		if err != nil {
			t.Fatal(err)
		}

		if output.Body.Message != "database connection OK" {
			t.Fatalf("got %q, want %q", output.Body.Message, "database connection OK")
		}
	})

	t.Run("a failing check returns a fixed 503 that keeps the cause", func(t *testing.T) {
		cause := errors.New("dial tcp 10.0.0.1:5432: connect: connection refused")

		_, err := GetDatabaseHealth(
			&dbtest.Stub{PingErr: cause},
		)(context.Background(), &GetHealthInput{})
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

		if strings.Contains(statusErr.Error(), "connection refused") {
			t.Fatalf("the response error leaks the cause: %q", statusErr.Error())
		}

		if !errors.Is(errors.Unwrap(err), cause) {
			t.Fatalf("got wrapped cause %v, want %v", errors.Unwrap(err), cause)
		}
	})
}

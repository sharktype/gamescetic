package health

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	dbtest "github.com/sharktype/satchel/internal/db/dbtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGetDatabaseHealth(t *testing.T) {
	t.Run("a reachable database reports OK", func(t *testing.T) {
		output, err := GetDatabaseHealth(zap.NewNop(), &dbtest.Stub{})(
			context.Background(), &GetHealthInput{},
		)
		if err != nil {
			t.Fatal(err)
		}

		if output.Body.Message != "database connection OK" {
			t.Fatalf("got %q, want %q", output.Body.Message, "database connection OK")
		}
	})

	t.Run("a failing check logs the cause and returns a fixed 503", func(t *testing.T) {
		cause := errors.New("dial tcp 10.0.0.1:5432: connect: connection refused")

		core, logs := observer.New(zapcore.DebugLevel)

		logger := zap.New(core)

		_, err := GetDatabaseHealth(
			logger, &dbtest.Stub{PingErr: cause},
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

		entries := logs.FilterMessage("database health check failed").All()
		if len(entries) != 1 {
			t.Fatalf("got %d matching log entries, want 1", len(entries))
		}

		if logged, _ := entries[0].ContextMap()["error"].(string); !strings.Contains(
			logged, "connection refused",
		) {
			t.Fatalf("the log entry lost the cause: %q", logged)
		}
	})
}

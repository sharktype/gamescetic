package health

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/clients"
	dbtest "github.com/sharktype/satchel/internal/db-test"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGetDatabaseHealth(t *testing.T) {
	t.Run("a reachable database reports OK", func(t *testing.T) {
		output, err := GetDatabaseHealth(zap.NewNop(), &clients.Clients{DB: &dbtest.Stub{}})(
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

		var logged bytes.Buffer

		logger := zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(&logged),
			zapcore.DebugLevel,
		))

		_, err := GetDatabaseHealth(
			logger, &clients.Clients{DB: &dbtest.Stub{PlaceholderErr: cause}},
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

		if !strings.Contains(logged.String(), "database health check failed") {
			t.Fatalf("the failure was not logged: %q", logged.String())
		}

		if !strings.Contains(logged.String(), "connection refused") {
			t.Fatalf("the log entry lost the cause: %q", logged.String())
		}
	})
}

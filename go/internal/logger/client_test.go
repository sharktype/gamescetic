package logger

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestAppLog(t *testing.T) {
	t.Run("logs the message and fields", func(t *testing.T) {
		logs := replaceGlobalLogger(t)

		Info("hello", zap.String("key", "value"))

		entries := logs.All()
		if len(entries) != 1 {
			t.Fatalf("got %d log entries, want 1", len(entries))
		}

		entry := entries[0]
		if entry.Level != zapcore.InfoLevel {
			t.Fatalf("got level %v, want %v", entry.Level, zapcore.InfoLevel)
		}

		if entry.Message != "hello" {
			t.Fatalf("got message %q, want %q", entry.Message, "hello")
		}

		if got := entry.ContextMap()["key"]; got != "value" {
			t.Fatalf("got field %v, want %v", got, "value")
		}
	})

	t.Run("reports the call site", func(t *testing.T) {
		logs := replaceGlobalLogger(t)

		Info("hello")

		entries := logs.All()
		if len(entries) != 1 {
			t.Fatalf("got %d log entries, want 1", len(entries))
		}

		if caller := entries[0].Caller; !strings.HasSuffix(caller.File, "client_test.go") {
			t.Fatalf("got caller %+v, want client_test.go", caller)
		}
	})
}

func TestInit(t *testing.T) {
	t.Run("installs a debug logger when verbose", func(t *testing.T) {
		previous := zap.L()
		t.Cleanup(func() { zap.ReplaceGlobals(previous) })

		shutdown, err := Init(Config{Verbose: true})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(shutdown)

		if !zap.L().Core().Enabled(zapcore.DebugLevel) {
			t.Fatal("debug logging is disabled")
		}
	})

	t.Run("installs an info logger by default", func(t *testing.T) {
		previous := zap.L()
		t.Cleanup(func() { zap.ReplaceGlobals(previous) })

		shutdown, err := Init(Config{})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(shutdown)

		if zap.L().Core().Enabled(zapcore.DebugLevel) {
			t.Fatal("debug logging is enabled")
		}

		if !zap.L().Core().Enabled(zapcore.InfoLevel) {
			t.Fatal("info logging is disabled")
		}
	})
}

func replaceGlobalLogger(t *testing.T) *observer.ObservedLogs {
	t.Helper()

	core, logs := observer.New(zapcore.InfoLevel)

	previous := zap.L()
	zap.ReplaceGlobals(zap.New(core, zap.AddCaller()))
	t.Cleanup(func() { zap.ReplaceGlobals(previous) })

	return logs
}

func TestEnableSentry(t *testing.T) {
	t.Run("logs carry the active OTel trace", func(t *testing.T) {
		transport := &sentry.MockTransport{}
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn:           "https://public@example.com/1",
			Transport:     transport,
			EnableTracing: true,
			Integrations: func(integrations []sentry.Integration) []sentry.Integration {
				return append(integrations, sentryotel.NewOtelIntegration())
			},
		})
		if err != nil {
			t.Fatalf("creating client: %v", err)
		}

		previousClient := sentry.CurrentHub().Client()
		t.Cleanup(func() { sentry.CurrentHub().BindClient(previousClient) })
		sentry.CurrentHub().BindClient(client)

		core, logs := observer.New(zapcore.InfoLevel)
		base := zap.New(core)

		previousLogger := zap.L()
		zap.ReplaceGlobals(base)

		t.Cleanup(func() {
			zap.ReplaceGlobals(previousLogger)
		})

		EnableSentry(context.Background())

		provider := sdktrace.NewTracerProvider()
		t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

		ctx, span := provider.Tracer("test").Start(context.Background(), "test")
		defer span.End()

		WithContext(ctx).Info("hello")

		if logs.Len() != 1 {
			t.Fatalf("got %d console entries, want 1", logs.Len())
		}

		sentry.Flush(time.Second)

		events := transport.Events()
		if len(events) != 1 || len(events[0].Logs) != 1 {
			t.Fatalf("got %d events, want one log event", len(events))
		}

		log := events[0].Logs[0]

		if log.Body != "hello" {
			t.Fatalf("got body %q, want %q", log.Body, "hello")
		}

		if got := log.TraceID; got != sentry.TraceID(span.SpanContext().TraceID()) {
			t.Errorf("got trace ID %v, want %s", got, span.SpanContext().TraceID())
		}

		if got := log.SpanID; got != sentry.SpanID(span.SpanContext().SpanID()) {
			t.Errorf("got span ID %v, want %s", got, span.SpanContext().SpanID())
		}
	})
}

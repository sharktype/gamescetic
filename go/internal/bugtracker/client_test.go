package bugtracker

import (
	"context"
	"errors"
	"testing"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestInit(t *testing.T) {
	t.Run("a valid DSN in production enables the bugtracker", func(t *testing.T) {
		previous := sentry.CurrentHub().Client()
		t.Cleanup(func() { sentry.CurrentHub().BindClient(previous) })

		shutdown, err := Init("https://public@example.com/1", true)
		if err != nil {
			t.Fatalf("got error %v, want none", err)
		}
		t.Cleanup(shutdown)

		if !sentry.CurrentHub().Client().IsEnabled() {
			t.Fatal("the bugtracker was not initialized")
		}
	})

	t.Run("an empty DSN disables the bugtracker", func(t *testing.T) {
		shutdown, err := Init("", true)
		if err != nil {
			t.Fatalf("got error %v, want none", err)
		}

		shutdown()
	})

	t.Run("a DSN outside production is ignored", func(t *testing.T) {
		sentry.CurrentHub().BindClient(nil)

		shutdown, err := Init("https://public@example.com/1", false)
		if err != nil {
			t.Fatalf("got error %v, want none", err)
		}

		shutdown()

		if sentry.CurrentHub().Client().IsEnabled() {
			t.Fatal("the bugtracker was initialized outside production")
		}
	})

	t.Run("an invalid DSN is rejected", func(t *testing.T) {
		shutdown, err := Init("not-a-dsn", true)
		if err == nil {
			t.Fatal("expected an error")
		}

		if shutdown != nil {
			t.Fatal("expected no shutdown function")
		}
	})
}

func TestReportError(t *testing.T) {
	t.Run("the captured event is linked to the active span", func(t *testing.T) {
		transport := &sentry.MockTransport{}
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn:       "https://public@example.com/1",
			Transport: transport,
			Integrations: func(integrations []sentry.Integration) []sentry.Integration {
				return append(integrations, sentryotel.NewOtelIntegration())
			},
		})
		if err != nil {
			t.Fatalf("creating client: %v", err)
		}

		previous := sentry.CurrentHub().Client()
		t.Cleanup(func() { sentry.CurrentHub().BindClient(previous) })
		sentry.CurrentHub().BindClient(client)

		provider := trace.NewTracerProvider()
		t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

		ctx, span := provider.Tracer("test").Start(context.Background(), "test")
		defer span.End()

		ReportError(ctx, "boom", errors.New("boom"))

		events := transport.Events()
		if len(events) != 1 {
			t.Fatalf("got %d events, want 1", len(events))
		}

		traceContext := events[0].Contexts["trace"]

		if got := traceContext["trace_id"]; got != span.SpanContext().TraceID().String() {
			t.Errorf("got trace ID %v, want %s", got, span.SpanContext().TraceID())
		}

		if got := traceContext["span_id"]; got != span.SpanContext().SpanID().String() {
			t.Errorf("got span ID %v, want %s", got, span.SpanContext().SpanID())
		}
	})
}

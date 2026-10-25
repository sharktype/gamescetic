package bugtracker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

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

		provider := sdktrace.NewTracerProvider()
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

func TestPanicRequest(t *testing.T) {
	t.Run("the panic is reported with its request and ID", func(t *testing.T) {
		transport := &sentry.MockTransport{}
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn:       "https://public@example.com/1",
			Transport: transport,
		})
		if err != nil {
			t.Fatalf("creating client: %v", err)
		}

		previous := sentry.CurrentHub().Client()
		t.Cleanup(func() { sentry.CurrentHub().BindClient(previous) })
		sentry.CurrentHub().BindClient(client)

		request := httptest.NewRequest(http.MethodGet, "/panic", nil)
		PanicRequest(request, "request-1", "arbitrary panic")

		events := transport.Events()
		if len(events) != 1 {
			t.Fatalf("got %d events, want 1", len(events))
		}

		event := events[0]

		if event.Request == nil || event.Request.Method != http.MethodGet {
			t.Fatalf("got request %+v, want the panicking request", event.Request)
		}

		if got := event.Tags["request_id"]; got != "request-1" {
			t.Errorf("got request ID %q, want %q", got, "request-1")
		}

		if len(event.Exception) != 1 || !strings.Contains(
			event.Exception[0].Value, "arbitrary panic",
		) {
			t.Fatalf("got exceptions %v, want the panic", event.Exception)
		}

		if event.Exception[0].Stacktrace == nil {
			t.Fatal("the event lost the stack trace")
		}
	})
}

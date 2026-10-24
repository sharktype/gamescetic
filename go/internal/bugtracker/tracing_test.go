package bugtracker

import (
	"context"
	"testing"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSentrySpanProcessor(t *testing.T) {
	t.Run("a root span becomes a transaction", func(t *testing.T) {
		transport, provider := newTestTracer(t)

		_, root := provider.Tracer("test").Start(context.Background(), "GET /things")
		root.SetAttributes(
			attribute.String("http.request.method", "GET"),
			attribute.String("url.path", "/things"),
		)
		root.End()

		event := transactionEvent(t, transport.Events())

		if got := event.Transaction; got != "GET /things" {
			t.Errorf("got transaction %q, want %q", got, "GET /things")
		}

		traceContext := event.Contexts["trace"]

		if got := traceContext["trace_id"]; got != sentry.TraceID(root.SpanContext().TraceID()) {
			t.Errorf("got trace ID %v, want %s", got, root.SpanContext().TraceID())
		}

		if got := traceContext["span_id"]; got != sentry.SpanID(root.SpanContext().SpanID()) {
			t.Errorf("got span ID %v, want %s", got, root.SpanContext().SpanID())
		}
	})

	t.Run("child spans are attached to the transaction", func(t *testing.T) {
		transport, provider := newTestTracer(t)

		ctx, root := provider.Tracer("test").Start(context.Background(), "GET /things")
		_, child := provider.Tracer("test").Start(ctx, "select things")
		child.SetAttributes(attribute.String("db.system", "postgresql"))
		child.End()
		root.End()

		event := transactionEvent(t, transport.Events())

		if len(event.Spans) != 1 {
			t.Fatalf("got %d spans, want 1", len(event.Spans))
		}

		span := event.Spans[0]

		if got := span.Description; got != "select things" {
			t.Errorf("got description %q, want %q", got, "select things")
		}

		if got := span.ParentSpanID; got != sentry.SpanID(root.SpanContext().SpanID()) {
			t.Errorf("got parent span ID %v, want %s", got, root.SpanContext().SpanID())
		}
	})
}

func newTestTracer(t *testing.T) (*sentry.MockTransport, *sdktrace.TracerProvider) {
	t.Helper()

	transport := &sentry.MockTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              "https://public@example.com/1",
		EnableTracing:    true,
		TracesSampleRate: 1.0,
		Transport:        transport,
	})
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}

	previous := sentry.CurrentHub().Client()
	t.Cleanup(func() { sentry.CurrentHub().BindClient(previous) })
	sentry.CurrentHub().BindClient(client)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(newSentrySpanProcessor()),
	)
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	return transport, provider
}

func transactionEvent(t *testing.T, events []*sentry.Event) *sentry.Event {
	t.Helper()

	for _, event := range events {
		if event.Type == "transaction" {
			return event
		}
	}

	t.Fatalf("got %d events, none of them transactions", len(events))

	return nil
}

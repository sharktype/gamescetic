package routing

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestTracing(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})

	t.Run("names matched routes", func(t *testing.T) {
		recorder.Reset()
		router := testRouter(t)
		router.Get("/traced", func(http.ResponseWriter, *http.Request) {})

		get(router, "/traced", "")

		spans := recorder.Ended()
		if len(spans) != 1 {
			t.Fatalf("got %d spans, want 1", len(spans))
		}

		if got := spans[0].Name(); got != "GET /traced" {
			t.Fatalf("got span %q, want %q", got, "GET /traced")
		}

		var route string
		for _, attr := range spans[0].Attributes() {
			if attr.Key == semconv.HTTPRouteKey {
				route = attr.Value.AsString()
			}
		}

		if route != "/traced" {
			t.Fatalf("got route %q, want %q", route, "/traced")
		}
	})

	t.Run("leaves unmatched routes unnamed", func(t *testing.T) {
		recorder.Reset()
		router := testRouter(t)
		router.Get("/traced", func(http.ResponseWriter, *http.Request) {})

		get(router, "/missing", "")

		spans := recorder.Ended()
		if len(spans) != 1 {
			t.Fatalf("got %d spans, want 1", len(spans))
		}

		if got := spans[0].Name(); got != "http.server" {
			t.Fatalf("got span %q, want %q", got, "http.server")
		}
	})
}

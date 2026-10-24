package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestInit(t *testing.T) {
	t.Run("is skipped without an endpoint", func(t *testing.T) {
		shutdown, err := Init(context.Background(), Config{ServiceName: "test"})
		if err != nil {
			t.Fatal(err)
		}

		shutdown()
	})

	t.Run("rejects an endpoint without a scheme", func(t *testing.T) {
		_, err := Init(context.Background(), Config{Endpoint: "localhost:4317"})
		if err == nil {
			t.Fatal("got no error")
		}
	})
}

func TestHTTPMiddleware(t *testing.T) {
	recorder := newSpanRecorder(t)

	handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/42", nil))

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	if got := spans[0].Name(); got != http.MethodGet {
		t.Fatalf("got span %q, want %q", got, http.MethodGet)
	}

	var path string
	for _, attr := range spans[0].Attributes() {
		if attr.Key == "url.path" {
			path = attr.Value.AsString()
		}
	}

	if path != "/items/42" {
		t.Fatalf("got path %q, want %q", path, "/items/42")
	}
}

func TestHTTPMiddlewareSkipsPaths(t *testing.T) {
	recorder := newSpanRecorder(t)

	handler := HTTPMiddleware("/ping")(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	if spans := recorder.Ended(); len(spans) != 0 {
		t.Fatalf("got %d spans, want none", len(spans))
	}
}

func newSpanRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})

	return recorder
}

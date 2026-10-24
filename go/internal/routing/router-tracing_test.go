package routing

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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

	t.Run("labels metrics with matched routes", func(t *testing.T) {
		reader := metric.NewManualReader()
		previous := otel.GetMeterProvider()
		otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(reader)))

		t.Cleanup(func() { otel.SetMeterProvider(previous) })

		router := testRouter(t)
		router.Get("/traced/{id}", func(http.ResponseWriter, *http.Request) {})

		get(router, "/traced/42", "")

		var collected metricdata.ResourceMetrics
		if err := reader.Collect(context.Background(), &collected); err != nil {
			t.Fatal(err)
		}

		if route := collectedRoute(t, collected); route != "/traced/{id}" {
			t.Fatalf("got route %q, want %q", route, "/traced/{id}")
		}
	})

	t.Run("leaves unmatched spans at the method name", func(t *testing.T) {
		recorder.Reset()
		router := testRouter(t)
		router.Get("/traced", func(http.ResponseWriter, *http.Request) {})

		get(router, "/missing", "")

		spans := recorder.Ended()
		if len(spans) != 1 {
			t.Fatalf("got %d spans, want 1", len(spans))
		}

		if got := spans[0].Name(); got != http.MethodGet {
			t.Fatalf("got span %q, want %q", got, http.MethodGet)
		}
	})
}

func collectedRoute(t *testing.T, collected metricdata.ResourceMetrics) string {
	t.Helper()

	for _, scope := range collected.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != "http.server.request.duration" {
				continue
			}

			histogram, ok := candidate.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatalf("metric %q is not a histogram", candidate.Name)
			}

			for _, point := range histogram.DataPoints {
				if route, ok := point.Attributes.Value(semconv.HTTPRouteKey); ok {
					return route.AsString()
				}
			}
		}
	}

	t.Fatal("no request duration metric with a route")

	return ""
}

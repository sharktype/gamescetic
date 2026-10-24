package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/sharktype/satchel/internal/logger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.uber.org/zap"
)

const (
	exportTimeout  = 5 * time.Second
	metricInterval = 30 * time.Second
)

func Init(ctx context.Context, cfg Config) (shutdown func(), err error) {
	if cfg.Endpoint == "" {
		logger.Info("no OTLP endpoint provided, skipping telemetry")

		return func() {}, nil
	}

	if err := validateEndpoint(cfg.Endpoint); err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentNameKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build telemetry resource: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpointURL(cfg.Endpoint),
		otlptracegrpc.WithHeaders(cfg.Headers),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	metricExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpointURL(cfg.Endpoint),
		otlpmetricgrpc.WithHeaders(cfg.Headers),
	)
	if err != nil {
		_ = traceExporter.Shutdown(ctx)

		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithInterval(metricInterval),
		)),
		metric.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Warn("telemetry export failed", zap.Error(err))
	}))

	logger.Info("opentelemetry initialized", zap.String("endpoint", cfg.Endpoint))

	return func() {
		shutdown := func(name string, stop func(context.Context) error) {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), exportTimeout)
			defer cancel()

			if err := stop(shutdownCtx); err != nil {
				logger.Warn(
					"failed to shut down telemetry",
					zap.String("provider", name),
					zap.Error(err),
				)
			}
		}

		shutdown("tracing", tracerProvider.Shutdown)
		shutdown("metrics", meterProvider.Shutdown)
	}, nil
}

// HTTPMiddleware instruments requests with spans and metrics, skipping requests in skipPaths.
func HTTPMiddleware(skipPaths ...string) func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware(
		"http.server",
		otelhttp.WithFilter(func(request *http.Request) bool {
			return !slices.Contains(skipPaths, request.URL.Path)
		}),
	)
}

func validateEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid OTLP endpoint %q: %w", endpoint, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("OTLP endpoint %q must use the http or https scheme", endpoint)
	}

	if parsed.Host == "" {
		return fmt.Errorf("OTLP endpoint %q must include a host", endpoint)
	}

	return nil
}

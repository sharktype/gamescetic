package bugtracker

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	sentryotlp "github.com/getsentry/sentry-go/otel/otlp"
	"github.com/sharktype/satchel/internal/logger"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

const (
	flushTimeout         = 2 * time.Second
	traceShutdownTimeout = 5 * time.Second
	traceSampleRate      = 1.0
)

func Init(dsn string, isProduction bool) (shutdown func(), err error) {
	if dsn == "" {
		logger.Info("no DSN provided, skipping bugtracker")

		return func() {}, nil
	}

	if !isProduction {
		logger.Warn("Sentry DSN provided outside production environment, skipping bugtracker")

		return func() {}, nil
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		EnableTracing:    true,
		TracesSampleRate: traceSampleRate,
		Integrations: func(integrations []sentry.Integration) []sentry.Integration {
			return append(integrations, sentryotel.NewOtelIntegration())
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sentry: %w", err)
	}

	// Captured errors are linked to the active span so the tracer provider must be global.

	exporter, err := sentryotlp.NewTraceExporter(context.Background(), dsn)
	if err != nil {
		sentry.Flush(flushTimeout)

		return nil, fmt.Errorf("failed to initialize trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(traceSampleRate))),
	)
	otel.SetTracerProvider(tracerProvider)

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), traceShutdownTimeout)
		defer cancel()

		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shut down tracer provider", zap.Error(err))
		}

		sentry.Flush(flushTimeout)
	}, nil
}

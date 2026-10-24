package bugtracker

import (
	"context"
	"fmt"
	"net/http"
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
		BeforeSend:       addSourceContext,
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

func ReportError(ctx context.Context, msg string, err error, fields ...zap.Field) {
	logger.Error(msg, append([]zap.Field{zap.Error(err)}, fields...)...)

	hub := sentry.CurrentHub()
	hub.Client().CaptureException(
		err,
		&sentry.EventHint{Context: ctx},
		hub.Scope(),
	)
}

func PanicRequest(request *http.Request, requestID string, recovered any) {
	logger.Error(fmt.Sprintf(
		"panic serving %s %s: %v", request.Method, request.URL.Path, recovered,
	))

	event := sentry.NewEvent()
	event.Level = sentry.LevelError
	event.Message = fmt.Sprintf("panic serving %s %s", request.Method, request.URL.Path)
	event.Request = sentry.NewRequest(request)
	event.Tags = map[string]string{"request_id": requestID}
	event.Exception = []sentry.Exception{{
		Type:       fmt.Sprintf("%T", recovered),
		Value:      fmt.Sprint(recovered),
		Stacktrace: sentry.NewStacktrace(),
	}}

	hub := sentry.CurrentHub()
	hub.Client().CaptureEvent(
		event,
		&sentry.EventHint{Context: request.Context()},
		hub.Scope(),
	)
}

package bugtracker

import (
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sharktype/satchel/internal/logger"
	"go.uber.org/zap"
)

const (
	flushTimeout    = 2 * time.Second
	traceSampleRate = 0.1
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
		TracesSampleRate: traceSampleRate,
		BeforeSend:       addSourceContext,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sentry: %w", err)
	}

	return func() {
		sentry.Flush(flushTimeout)
	}, nil
}

func ReportError(msg string, err error, fields ...zap.Field) {
	logger.Error(msg, append([]zap.Field{zap.Error(err)}, fields...)...)

	sentry.CaptureException(err)
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

	sentry.CaptureEvent(event)
}

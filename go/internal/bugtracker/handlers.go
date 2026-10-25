package bugtracker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/sharktype/satchel/internal/logger"
	"go.uber.org/zap"
)

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

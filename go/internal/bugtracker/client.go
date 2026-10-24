package bugtracker

import (
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sharktype/satchel/internal/logger"
	"go.uber.org/zap"
)

const flushTimeout = 2 * time.Second

func Init(dsn string) (shutdown func()) {
	if dsn == "" {
		logger.Info("no DSN provided, skipping bugtracker")

		return func() {}
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		TracesSampleRate: 0.1,
	})
	if err != nil {
		logger.Fatal("failed to initialize bugtracker", zap.Error(err))
	}

	return func() {
		sentry.Flush(flushTimeout)
	}
}

func Notify(err error) {
	sentry.CaptureException(err)
}

package bugtracker

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

const (
	flushTimeout = 2 * time.Second
	sampleRate   = 0.1
)

func Init(logger *zap.Logger, dsn string) (shutdown func()) {
	if dsn == "" {
		logger.Info("no DSN provided, skipping bugtracker")

		return func() {}
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		TracesSampleRate: sampleRate,
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

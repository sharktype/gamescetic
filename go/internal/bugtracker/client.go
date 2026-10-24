package bugtracker

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sharktype/satchel/internal/logger"
)

const (
	flushTimeout    = 2 * time.Second
	traceSampleRate = 0.1
)

func Init(dsn string) (shutdown func(), err error) {
	if dsn == "" {
		logger.Info("no DSN provided, skipping bugtracker")

		return func() {}, nil
	}

	err = sentry.Init(sentry.ClientOptions{Dsn: dsn, TracesSampleRate: traceSampleRate})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize sentry: %w", err)
	}

	return func() {
		sentry.Flush(flushTimeout)
	}, nil
}

func Report(err error) {
	logger.Error(err.Error())
	sentry.CaptureException(err)
}

package logger

import (
	"context"
	"log"

	"github.com/getsentry/sentry-go"
	sentryzap "github.com/getsentry/sentry-go/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init installs the application logger globally and returns a disposal function.
func Init(cfg Config) (shutdown func(), err error) {
	level := zap.InfoLevel
	if cfg.Verbose {
		level = zap.DebugLevel
	}

	config := zap.NewDevelopmentConfig()
	if cfg.Production {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(level)

	built, err := config.Build()
	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(built)

	return func() { _ = zap.L().Sync() }, nil
}

// EnableSentry tees a Sentry log core into the global logger.
//
// It must run after Sentry has been initialised.
func EnableSentry(ctx context.Context) {
	if !sentry.CurrentHub().Client().IsEnabled() {
		return
	}

	core := sentryzap.NewSentryCore(ctx, sentryzap.Option{
		Level: []zapcore.Level{
			zapcore.InfoLevel,
			zapcore.WarnLevel,
			zapcore.ErrorLevel,
		},
		AddCaller: true,
	})

	zap.ReplaceGlobals(zap.L().WithOptions(zap.WrapCore(func(existing zapcore.Core) zapcore.Core {
		return zapcore.NewTee(existing, core)
	})))
}

// WithContext returns a logger whose entries are correlated with the active span in ctx.
func WithContext(ctx context.Context) *zap.Logger {
	return appLog().With(sentryzap.Context(ctx))
}

func Debug(msg string, fields ...zap.Field) {
	appLog().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	appLog().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	appLog().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	appLog().Error(msg, fields...)
}

func StdLogger() *log.Logger {
	return zap.NewStdLog(zap.L())
}

// appLog skips the wrapper frame so callers report their real source location.
func appLog() *zap.Logger {
	return zap.L().WithOptions(zap.AddCallerSkip(1))
}

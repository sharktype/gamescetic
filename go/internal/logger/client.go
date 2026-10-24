package logger

import (
	"log"

	"go.uber.org/zap"
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

	return func() { _ = built.Sync() }, nil
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

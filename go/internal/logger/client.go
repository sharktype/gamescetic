package logger

import (
	"go.uber.org/zap"
)

var globalLogger = zap.NewNop()

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

	globalLogger = built

	return func() { _ = built.Sync() }, nil
}

// Set installs an already-built logger.
//
// Tests use it to capture output. Don't use it in production logic.
func Set(built *zap.Logger) {
	globalLogger = built
}

// Global returns the global logger directly.
func Global() *zap.Logger {
	return globalLogger
}

// Re-export to simplify the logging API in the application:

func Debug(msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	globalLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	globalLogger.Error(msg, fields...)
}

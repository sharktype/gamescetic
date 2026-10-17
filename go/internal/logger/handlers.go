package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func MustNew() *zap.Logger {
	logger, err := newLogger()
	if err != nil {
		panic(fmt.Sprintf("error creating logger: %v", err))
	}

	return logger
}

func newLogger() (*zap.Logger, error) {
	level := zap.InfoLevel
	if verbose := os.Getenv("VERBOSE"); verbose != "" {
		level = zap.DebugLevel
	}

	config := zap.NewDevelopmentConfig()
	if os.Getenv("PRODUCTION") != "" {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(level)

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	defer func(logger *zap.Logger) {
		// Flush the logger to ensure all logs are written before the program exits.

		_ = logger.Sync()
	}(logger)

	return logger, nil
}

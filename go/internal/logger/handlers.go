package logger

import (
	"os"

	"go.uber.org/zap"
)

func New() (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	level := zap.InfoLevel
	if verbose := os.Getenv("VERBOSE"); verbose != "" {
		level = zap.DebugLevel
	}

	if os.Getenv("PRODUCTION") == "" {
		logger, err = zap.NewDevelopment(zap.IncreaseLevel(level))
		if err != nil {
			return nil, err
		}
	} else {
		logger, err = zap.NewProduction(zap.IncreaseLevel(level))
		if err != nil {
			return nil, err
		}
	}

	defer func(logger *zap.Logger) {
		// Flush the logger to ensure all logs are written before the program exits.

		_ = logger.Sync()
	}(logger)

	return logger, nil
}

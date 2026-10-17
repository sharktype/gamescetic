package logger

import (
	"os"

	"github.com/sharktype/satchel/internal/utils"
	"go.uber.org/zap"
)

func MustNew() *zap.Logger {
	logger, err := newLogger()
	utils.Must(err, "error creating logger")

	return logger
}

func newLogger() (*zap.Logger, error) {
	level := zap.InfoLevel
	if verbose := os.Getenv("VERBOSE"); verbose != "" {
		level = zap.DebugLevel
	}

	config := zap.NewDevelopmentConfig()
	if utils.IsProduction() {
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

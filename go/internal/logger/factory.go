package logger

import (
	"github.com/sharktype/satchel/internal/utils"
	"go.uber.org/zap"
)

func MustNew(cfg Config) *zap.Logger {
	logger, err := newLogger(cfg)
	utils.Must(err, "error creating logger")

	return logger
}

func newLogger(cfg Config) (*zap.Logger, error) {
	level := zap.InfoLevel
	if cfg.Verbose {
		level = zap.DebugLevel
	}

	config := zap.NewDevelopmentConfig()
	if cfg.Production {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(level)

	return config.Build()
}

package logger

import (
	"go.uber.org/zap"
)

func New(cfg Config) (*zap.Logger, error) {
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

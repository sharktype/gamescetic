package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func MustNew(logger *zap.Logger) *Config {
	cfg, err := newConfig(logger)
	if err != nil {
		panic(fmt.Sprintf("error loading config: %v", err))
	}

	return cfg
}

func newConfig(logger *zap.Logger) (*Config, error) {
	if os.Getenv("PRODUCTION") == "" {
		logger.Info("app booting in development mode...")

		if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	} else {
		logger.Info("app booting in production mode...")
	}

	cfg := new(Config)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

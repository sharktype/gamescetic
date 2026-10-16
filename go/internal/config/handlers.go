package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func New(logger *zap.Logger) (*Config, error) {
	if os.Getenv("PRODUCTION") == "" {
		logger.Info("app booting in development mode...")

		err := godotenv.Load(".env")
		if err != nil {
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

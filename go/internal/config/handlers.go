package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/sharktype/satchel/internal/utils"
	"go.uber.org/zap"
)

func MustNew(logger *zap.Logger) *Config {
	cfg, err := newConfig(logger)

	utils.Must(err, "error loading config")

	return cfg
}

func newConfig(logger *zap.Logger) (*Config, error) {
	if !utils.IsProduction() {
		logger.Info("app booting in development mode...")

		if err := loadDotEnv(); err != nil {
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

// loadDotEnv loads envars anchored on go.mod's location.
//
// Remember that dotenv loading is only done in development; prod must specify envars more
// explicitly on the environment and don't run into this problem.
func loadDotEnv() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			err := godotenv.Load(filepath.Join(dir, ".env"))
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}

			return err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}

		dir = parent
	}
}

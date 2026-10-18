package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

func New() (*Config, error) {
	production, err := isProduction()
	if err != nil {
		return nil, err
	}

	if !production {
		if err := loadDotEnv(); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	cfg := new(Config)
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	cfg.Production = production

	return cfg, nil
}

func isProduction() (bool, error) {
	production, isSet := os.LookupEnv("PRODUCTION")
	if !isSet {
		return false, nil
	}

	if production != "1" {
		return false, fmt.Errorf(`PRODUCTION must be unset or "1", got %q`, production)
	}

	return true, nil
}

// loadDotEnv loads envars anchored on go.mod's location.
//
// Remember that dotenv loading is only done in development; prod must specify envars more
// explicitly on the environment and don't run into this problem.
func loadDotEnv() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
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

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

func New() (*Config, error) {
	production, err := IsProduction()
	if err != nil {
		return nil, err
	}

	if !production {
		if err := LoadEnv(); err != nil {
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

func IsProduction() (bool, error) {
	production, isSet := os.LookupEnv("PRODUCTION")
	if !isSet {
		return false, nil
	}

	if production != "1" {
		return false, fmt.Errorf(`PRODUCTION must be unset or "1", got %q`, production)
	}

	return true, nil
}

func LoadEnv() error {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("failed to locate .env")
	}

	err := godotenv.Load(filepath.Join(filepath.Dir(file), "..", "..", ".env"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func MustNewPool(ctx context.Context, cfg *Config, logger *zap.Logger) *pgxpool.Pool {
	pool, err := newPool(ctx, cfg, logger)
	if err != nil {
		panic(fmt.Sprintf("error creating database pool: %v", err))
	}

	return pool
}

func newPool(ctx context.Context, cfg *Config, logger *zap.Logger) (*pgxpool.Pool, error) {
	if cfg == nil || cfg.Driver == nil || cfg.DSN == nil {
		return nil, errors.New("database connection is required")
	}

	if *cfg.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver: %s", *cfg.Driver)
	}

	logger.Debug("connecting to postgres database and testing connection...")
	pool, err := pgxpool.New(ctx, *cfg.DSN)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("postgres database connection successful")

	return pool, nil
}

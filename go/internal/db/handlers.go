package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/internal/utils"
	"go.uber.org/zap"
)

func MustNewPool(ctx context.Context, logger *zap.Logger, cfg Config) *pgxpool.Pool {
	pool, err := newPool(ctx, logger, cfg)

	utils.Must(err, "error creating database pool")

	return pool
}

func newPool(ctx context.Context, logger *zap.Logger, cfg Config) (*pgxpool.Pool, error) {
	if cfg.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	logger.Debug("connecting to database and testing connection...")

	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("database connection successful")

	return pool, nil
}

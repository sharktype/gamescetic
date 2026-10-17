package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// driver is the only supported database driver.
const driver = "postgres"

func NewPool(ctx context.Context, logger *zap.Logger, cfg Config) (*pgxpool.Pool, error) {
	logger.Debug("connecting to database and testing connection...", zap.String("driver", driver))

	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("database connection successful", zap.String("driver", driver))

	return pool, nil
}

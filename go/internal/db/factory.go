package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// driver is the only supported database driver.
const driver = "postgres"

func NewPool(ctx context.Context, logger *zap.Logger, cfg Config) (*pgxpool.Pool, error) {
	logger.Debug("connecting to database and testing connection...", zap.String("driver", driver))

	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to ping database after connection: %w", err)
	}

	logger.Info("database connection successful", zap.String("driver", driver))

	return pool, nil
}

package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	driver         = "postgres"
	startupTimeout = 10 * time.Second
)

func NewPool(ctx context.Context, logger *zap.Logger, cfg Config) (*pgxpool.Pool, error) {
	logger.Debug("connecting to database and testing connection...", zap.String("driver", driver))

	startupCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	pool, err := pgxpool.New(startupCtx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	if err := pool.Ping(startupCtx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to ping database after connection: %w", err)
	}

	logger.Info("database connection successful", zap.String("driver", driver))

	return pool, nil
}

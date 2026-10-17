package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/internal/utils"
	"go.uber.org/zap"
)

// driver is the only supported database driver.
const driver = "postgres"

func MustNewPool(ctx context.Context, logger *zap.Logger, cfg Config) *pgxpool.Pool {
	pool, err := newPool(ctx, logger, cfg)

	utils.Must(err, "error creating database pool")

	return pool
}

func newPool(ctx context.Context, logger *zap.Logger, cfg Config) (*pgxpool.Pool, error) {
	logger.Debug(fmt.Sprintf("connecting to %s and testing connection...", driver))

	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("%s connection successful", driver))

	return pool, nil
}

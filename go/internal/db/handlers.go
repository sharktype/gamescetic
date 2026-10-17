package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/internal/utils"
	"go.uber.org/zap"
)

type (
	Driver string
	DSN    string
)

func MustNewPool(ctx context.Context, logger *zap.Logger, driver Driver, dsn DSN) *pgxpool.Pool {
	pool, err := newPool(ctx, logger, driver, dsn)

	utils.Must(err, "error creating database pool")

	return pool
}

func newPool(
	ctx context.Context, logger *zap.Logger, driver Driver, dsn DSN,
) (*pgxpool.Pool, error) {
	if driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	logger.Debug("connecting to database and testing connection...")

	pool, err := pgxpool.New(ctx, string(dsn))
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("database connection successful")

	return pool, nil
}

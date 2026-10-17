package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/db"
	sqlc "github.com/sharktype/satchel/db"
	"go.uber.org/zap"
)

// store implements the Client interface.
type store struct {
	*db.Queries
	db   pool
	inTx bool
}

type pool interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func New(ctx context.Context, cfg *Config, logger *zap.Logger) (Client, error) {
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

	return &store{
		Queries: sqlc.New(pool),
		db:      pool,
	}, nil
}

func (s *store) IsTx() bool {
	return s.inTx
}

func (s *store) ExecTx(ctx context.Context, fn func(Client) error) error {
	if s.inTx {
		return fn(s)
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})

	if err != nil {
		return err
	}

	defer func() {
		// Safe to ignore; actual failures kill the connection.

		_ = tx.Rollback(ctx)
	}()

	txClient := &store{
		Queries: s.WithTx(tx),
		db:      s.db,
		inTx:    true,
	}

	if err := fn(txClient); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

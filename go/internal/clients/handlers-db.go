package clients

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/db"
	"go.uber.org/zap"
)

type Pool interface {
	db.DBTX

	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type Store struct {
	*db.Queries
	db   Pool
	inTx bool
}

func New(ctx context.Context, cfg *Config, logger *zap.Logger) (*Clients, error) {
	database, err := setUpDB(ctx, cfg.DB, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("postgres database connection successful")

	appClients := &Clients{
		DB: newStore(database),
	}

	return appClients, nil
}

func (s *Store) IsTx() bool {
	return s.inTx
}

// ExecTx runs fn within a transaction.
//
// If the Store is already transactional, the callback runs directly without nesting.
func (s *Store) ExecTx(ctx context.Context, fn func(*Store) error) error {
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

	txStore := &Store{
		Queries: s.WithTx(tx),
		db:      s.db,
		inTx:    true,
	}

	if err := fn(txStore); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func newStore(database Pool) *Store {
	return &Store{
		Queries: db.New(database),
		db:      database,
	}
}

func setUpDB(ctx context.Context, cfg *DB, logger *zap.Logger) (*pgxpool.Pool, error) {
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

	return pool, nil
}

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/generated"
)

type Store struct {
	*generated.Queries
	pool *pgxpool.Pool
}

var _ Client = (*Store)(nil)

func New(pool *pgxpool.Pool) *Store {
	return &Store{
		Queries: generated.New(pool),
		pool:    pool,
	}
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// ExecTx runs fn in a transaction, committing only if fn returns nil.
func (s *Store) ExecTx(ctx context.Context, fn func(generated.Querier) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		// Safe to ignore; actual failures kill the connection.

		_ = tx.Rollback(context.WithoutCancel(ctx))
	}()

	if err := fn(s.WithTx(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

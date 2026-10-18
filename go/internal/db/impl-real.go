package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/generated"
)

// store implements the Client interface.
type store struct {
	*generated.Queries
	pool *pgxpool.Pool
	inTx bool
}

func New(pool *pgxpool.Pool) Client {
	return &store{
		Queries: generated.New(pool),
		pool:    pool,
	}
}

func (s *store) IsTx() bool {
	return s.inTx
}

func (s *store) Close() {
	s.pool.Close()
}

func (s *store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *store) ExecTx(ctx context.Context, fn func(Client) error) error {
	if s.inTx {
		return fn(s)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		// Safe to ignore; actual failures kill the connection.

		_ = tx.Rollback(ctx)
	}()

	txClient := &store{
		Queries: s.WithTx(tx),
		pool:    s.pool,
		inTx:    true,
	}

	if err := fn(txClient); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/generated"
)

// store implements the Client interface.
type store struct {
	*generated.Queries
	db   pool
	inTx bool
}

type pool interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

func MustNew(pool *pgxpool.Pool) Client {
	return &store{
		Queries: generated.New(pool),
		db:      pool,
	}
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

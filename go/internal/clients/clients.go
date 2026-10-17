package clients

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

// Intentional: Clients only wraps DB today, but it is the single place future clients (telemetry,
// Redis, third-party APIs, and so on) will be added, so it is not a pointless one-field wrapper.

type Clients struct {
	DB db.Client

	pool *pgxpool.Pool
}

func New(ctx context.Context, logger *zap.Logger, cfg db.Config) (*Clients, error) {
	pool, err := db.NewPool(ctx, logger, cfg)
	if err != nil {
		return nil, err
	}

	return &Clients{DB: db.New(pool), pool: pool}, nil
}

func (c *Clients) Close() {
	c.pool.Close()
}

package clients

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

type Clients struct {
	DB   db.Client
	Auth *auth.Client

	pool *pgxpool.Pool
}

func New(
	ctx context.Context, logger *zap.Logger, dbCfg db.Config, authCfg auth.Config,
) (*Clients, error) {
	pool, err := db.NewPool(ctx, logger, dbCfg)
	if err != nil {
		return nil, err
	}

	database := db.New(pool)

	authClient, err := auth.New(ctx, database, logger, authCfg)
	if err != nil {
		pool.Close()

		return nil, err
	}

	return &Clients{DB: database, Auth: authClient, pool: pool}, nil
}

func (c *Clients) Close() {
	c.pool.Close()
}

package clients

import (
	"context"
	"fmt"

	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

type Clients struct {
	DB   db.Client
	Auth *auth.Client
}

func New(
	ctx context.Context, logger *zap.Logger, dbCfg db.Config, authCfg auth.Config,
) (*Clients, error) {
	pool, err := db.NewPool(ctx, logger, dbCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	database := db.New(pool)

	authClient, err := auth.New(ctx, database, logger, authCfg)
	if err != nil {
		database.Close()

		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	return &Clients{DB: database, Auth: authClient}, nil
}

func (c *Clients) Close() {
	c.DB.Close()
}

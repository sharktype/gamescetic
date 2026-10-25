package clients

import (
	"context"
	"fmt"

	"github.com/sharktype/gamescetic/internal/auth"
	"github.com/sharktype/gamescetic/internal/db"
)

type Clients struct {
	DB   *db.Store
	Auth *auth.Client
}

func New(ctx context.Context, dbCfg db.Config, authCfg auth.Config) (*Clients, error) {
	pool, err := db.NewPool(ctx, dbCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	database := db.New(pool)

	authClient, err := auth.New(ctx, database, authCfg)
	if err != nil {
		database.Close()

		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	return &Clients{DB: database, Auth: authClient}, nil
}

func (c *Clients) Close() {
	c.Auth.Close()
	c.DB.Close()
}

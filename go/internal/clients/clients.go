package clients

import (
	"context"
	"fmt"

	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

type Clients struct {
	DB db.Client
}

func MustNew(ctx context.Context, cfg *Config, logger *zap.Logger) *Clients {
	appClients, err := new(ctx, cfg, logger)
	if err != nil {
		panic(fmt.Sprintf("error creating clients: %v", err))
	}

	return appClients
}

func new(ctx context.Context, cfg *Config, logger *zap.Logger) (*Clients, error) {
	database, err := db.New(ctx, cfg.DB, logger)
	if err != nil {
		return nil, err
	}

	return &Clients{DB: database}, nil
}

func NewStubs() (*Clients, error) {
	return &Clients{DB: db.NewStub()}, nil
}

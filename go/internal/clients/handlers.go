package clients

import (
	"context"

	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

func New(ctx context.Context, cfg *Config, logger *zap.Logger) (*Clients, error) {
	database, err := db.New(ctx, cfg.DB, logger)
	if err != nil {
		return nil, err
	}

	return &Clients{DB: database}, nil
}

func NewStubs() (*Clients, error) {
	return &Clients{DB: db.NewStub()}, nil
}

package clients

import (
	"context"

	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

type Clients struct {
	DB db.Client
}

func MustNew(ctx context.Context, cfg *Config, logger *zap.Logger) *Clients {
	return &Clients{DB: db.MustNew(db.MustNewPool(ctx, cfg.DB, logger))}
}

func NewStubs() *Clients {
	return &Clients{DB: db.NewStub()}
}

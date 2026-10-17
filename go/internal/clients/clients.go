package clients

import (
	"context"

	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

type (
	Driver string
	DSN    string
)

type Clients struct {
	DB db.Client
}

func MustNew(ctx context.Context, logger *zap.Logger, driver Driver, dsn DSN) *Clients {
	return &Clients{DB: db.MustNew(db.MustNewPool(ctx, string(driver), string(dsn), logger))}
}

func NewStubs() *Clients {
	return &Clients{DB: db.NewStub()}
}

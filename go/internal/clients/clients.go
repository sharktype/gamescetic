package clients

import (
	"context"

	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

type Clients struct {
	DB db.Client
}

func MustNew(ctx context.Context, logger *zap.Logger, driver, dsn string) *Clients {
	return &Clients{DB: db.MustNew(db.MustNewPool(ctx, logger, db.Driver(driver), db.DSN(dsn)))}
}

func NewStubs() *Clients {
	return &Clients{DB: db.NewStub()}
}

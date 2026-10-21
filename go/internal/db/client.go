package db

import (
	"context"

	"github.com/sharktype/satchel/generated"
)

type Client interface {
	generated.Querier

	ExecTx(context.Context, func(generated.Querier) error) error
	Ping(context.Context) error
	Close()
}

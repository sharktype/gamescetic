package db

import (
	"context"

	"github.com/sharktype/satchel/generated"
)

type Client interface {
	generated.Querier

	IsTx() bool
	ExecTx(context.Context, func(Client) error) error
}

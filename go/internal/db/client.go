package db

import (
	"context"

	"github.com/sharktype/satchel/generated"
)

type Client interface {
	generated.Querier

	// Intentional: IsTX and ExecTx cannot be deleted even if they're dead code.

	IsTx() bool
	ExecTx(context.Context, func(Client) error) error
	Ping(context.Context) error
}

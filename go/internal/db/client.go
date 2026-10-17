package db

import (
	"context"

	"github.com/sharktype/satchel/db"
)

type Client interface {
	db.Querier

	IsTx() bool
	ExecTx(context.Context, func(Client) error) error
}

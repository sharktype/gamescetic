package db

import (
	"context"

	"github.com/sharktype/satchel/generated"
)

type Client interface {
	generated.Querier

	Ping(context.Context) error
	Close()
}

package db_test

import (
	"context"

	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/db"
)

type Stub struct {
	db.Client

	User    generated.User
	UserErr error

	PingErr error
}

var _ db.Client = (*Stub)(nil)

func (s *Stub) GetUserBySub(context.Context, string) (generated.User, error) {
	return s.User, s.UserErr
}

func (s *Stub) Ping(context.Context) error {
	return s.PingErr
}

package dbtest

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/db"
)

type Stub struct {
	User    generated.User
	UserErr error

	PingErr error
}

var _ db.Client = (*Stub)(nil)

var errUnsupported = errors.New("dbtest: method not supported by Stub")

func ProvisionedUser() generated.User {
	return generated.User{
		ID:    "user-1",
		Sub:   "sub-1",
		Name:  "Ada Lovelace",
		Email: "ada-lovelace@example.com",
	}
}

func (s *Stub) GetUserBySub(_ context.Context, sub string) (generated.User, error) {
	if s.UserErr != nil {
		return generated.User{}, s.UserErr
	}

	if sub != s.User.Sub {
		return generated.User{}, pgx.ErrNoRows
	}

	return s.User, nil
}

func (s *Stub) GetMeta(context.Context, string) (string, error) {
	return "", errUnsupported
}

func (s *Stub) SetMeta(context.Context, generated.SetMetaParams) error {
	return errUnsupported
}

func (s *Stub) IsTx() bool {
	return false
}

func (s *Stub) ExecTx(context.Context, func(db.Client) error) error {
	return errUnsupported
}

func (s *Stub) Ping(context.Context) error {
	return s.PingErr
}

func (s *Stub) Close() {}

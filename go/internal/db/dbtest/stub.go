package dbtest

import (
	"context"

	"github.com/jackc/pgx/v5"
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

var ProvisionedUser = generated.User{
	ID:    "user-1",
	Sub:   "sub-1",
	Name:  "Ada Lovelace",
	Email: "ada-lovelace@example.com",
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

func (s *Stub) Ping(context.Context) error {
	return s.PingErr
}

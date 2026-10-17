package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/db"
)

var ErrUserNotProvisioned = errors.New("user is not provisioned")

type Manager struct {
	DB db.Client
}

func NewManager(client db.Client) *Manager {
	return &Manager{DB: client}
}

func (m *Manager) RequireUser(ctx context.Context, sub string) (generated.User, error) {
	user, err := m.DB.GetUserBySub(ctx, sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return generated.User{}, ErrUserNotProvisioned
	}

	if err != nil {
		return generated.User{}, fmt.Errorf("failed to look up user %q: %w", sub, err)
	}

	return user, nil
}

package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	sqlc "github.com/sharktype/satchel/db"
	"github.com/sharktype/satchel/internal/db"
)

var ErrUserNotProvisioned = errors.New("user is not provisioned")

type Manager struct {
	DB db.Client
}

func NewManager(client db.Client) *Manager {
	return &Manager{DB: client}
}

func (m *Manager) RequireUser(ctx context.Context, sub string) (sqlc.User, error) {
	row, err := m.DB.GetUserBySub(ctx, sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.User{}, ErrUserNotProvisioned
	}

	if err != nil {
		return sqlc.User{}, fmt.Errorf("failed to look up user %q: %w", sub, err)
	}

	return row.User, nil
}

func (m *Manager) FindViewerID(ctx context.Context) (string, bool) {
	user, ok := ViewerFromContext(ctx)
	if !ok {
		return "", false
	}

	return user.ID, true
}

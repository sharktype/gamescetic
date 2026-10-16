package auth

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/sharktype/satchel/db"
	"github.com/sharktype/satchel/internal/clients"
)

type Manager struct {
	DB *clients.Store

	cache map[string]struct{}
	mutex sync.RWMutex
}

func NewManager(store *clients.Store) *Manager {
	return &Manager{
		DB:    store,
		cache: make(map[string]struct{}),
	}
}

func (m *Manager) EnsureUser(ctx context.Context, rawToken string) error {
	parsed, err := jwt.ParseInsecure([]byte(rawToken))
	if err != nil {
		return err
	}

	sub, ok := parsed.Subject()
	if !ok || sub == "" {
		return errors.New("JWT is missing subject claim")
	}

	if m.isCached(sub) {
		return nil
	}

	if _, err := m.DB.GetUserBySub(ctx, sub); err == nil {
		m.addToCache(sub)

		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	email, name, avatarURL, err := parseUserClaims(parsed)
	if err != nil {
		return err
	}

	_, err = m.DB.CreateUser(ctx, db.CreateUserParams{
		ID:        uuid.New().String(),
		Sub:       sub,
		Name:      name,
		Email:     email,
		AvatarUrl: avatarURL,
	})
	if err != nil {
		// A concurrent request may have created the user between our SELECT and INSERT.

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			m.addToCache(sub)

			return nil
		}

		return err
	}

	m.addToCache(sub)

	return nil
}

func (m *Manager) FindViewerID(ctx context.Context) (string, bool) {
	sub, ok := SubjectFromContext(ctx)
	if !ok {
		return "", false
	}

	user, err := m.DB.GetUserBySub(ctx, sub)
	if err != nil {
		return "", false
	}

	return user.User.ID, true
}

func (m *Manager) isCached(id string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, ok := m.cache[id]

	return ok
}

func (m *Manager) addToCache(id string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cache[id] = struct{}{}
}

func parseUserClaims(parsed jwt.Token) (email, name string, avatarURL *string, err error) {
	if err := parsed.Get("email", &email); err != nil || email == "" {
		return "", "", nil, errors.New("JWT is missing email claim")
	}

	var userMetadata map[string]any
	if parsed.Get("user_metadata", &userMetadata) == nil {
		if v, ok := userMetadata["name"].(string); ok {
			name = v
		}

		if v, ok := userMetadata["avatar_url"].(string); ok && v != "" {
			avatarURL = &v
		}
	}

	return email, name, avatarURL, nil
}

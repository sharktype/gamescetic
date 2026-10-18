package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

const jwksMinInterval = 10 * time.Minute

// Client verifies tokens against the IDP's published keys and resolves the caller to a user.
type Client struct {
	cache  *jwk.Cache
	db     db.Client
	cfg    Config
	logger *zap.Logger
}

func New(ctx context.Context, db db.Client, logger *zap.Logger, cfg Config) (*Client, error) {
	cache, err := jwk.NewCache(ctx, httprc.NewClient())
	if err != nil {
		return nil, fmt.Errorf("failed to create jwk cache: %w", err)
	}

	err = cache.Register(ctx, cfg.JWKSURL, jwk.WithMinInterval(jwksMinInterval))
	if err != nil {
		return nil, fmt.Errorf("failed to register jwk location: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = cache.Refresh(fetchCtx, cfg.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch on startup: %w", err)
	}

	return &Client{cache: cache, cfg: cfg, db: db, logger: logger}, nil
}

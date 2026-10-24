package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jwx-go/jwkfetch/v4"
	"github.com/lestrrat-go/httprc/v3"
	"github.com/sharktype/satchel/internal/db"
	"go.uber.org/zap"
)

const jwksMinInterval = 10 * time.Minute

// Client verifies tokens against the IDP's published keys and resolves the caller to a user.
type Client struct {
	cache *jwkfetch.Cache

	db     *db.Store
	logger *zap.Logger
	cfg    Config

	cancel context.CancelFunc
}

func New(ctx context.Context, db *db.Store, logger *zap.Logger, cfg Config) (*Client, error) {
	cacheCtx, cancelCache := context.WithCancel(ctx)

	cache, err := jwkfetch.NewCache(cacheCtx, httprc.NewClient())
	if err != nil {
		cancelCache()

		return nil, fmt.Errorf("failed to create jwk cache: %w", err)
	}

	err = cache.Register(cacheCtx, cfg.JWKSURL, jwkfetch.WithMinInterval(jwksMinInterval))
	if err != nil {
		cancelCache()

		return nil, fmt.Errorf("failed to register jwk location: %w", err)
	}

	fetchCtx, cancelFetch := context.WithTimeout(cacheCtx, 10*time.Second)
	defer cancelFetch()

	_, err = cache.Refresh(fetchCtx, cfg.JWKSURL)
	if err != nil {
		cancelCache()

		return nil, fmt.Errorf("failed to fetch on startup: %w", err)
	}

	return &Client{
		cache:  cache,
		cancel: cancelCache,
		cfg:    cfg,
		db:     db,
		logger: logger,
	}, nil
}

// Close stops background key refreshes.
//
// Should only be called after in-flight requests have drained.
func (c *Client) Close() {
	c.cancel()
}

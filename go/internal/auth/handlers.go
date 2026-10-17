package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/sharktype/satchel/db"
)

const DefaultJWTSchemeKey = "JWT"

func NewJWKSet(ctx context.Context, jwkUrl string) (*jwk.Cache, error) {
	httpClient := httprc.NewClient()

	jwkCache, err := jwk.NewCache(ctx, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwk cache: %w", err)
	}

	err = jwkCache.Register(ctx, jwkUrl, jwk.WithMinInterval(10*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("failed to register jwk location: %w", err)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = jwkCache.Refresh(fetchCtx, jwkUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch on startup: %w", err)
	}

	return jwkCache, nil
}

func SubjectFromContext(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(subjectKey).(string)

	return s, ok && s != ""
}

func ViewerFromContext(ctx context.Context) (db.User, bool) {
	user, ok := ctx.Value(userKey).(db.User)

	return user, ok
}

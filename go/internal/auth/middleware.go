package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"go.uber.org/zap"
)

const JWTSchemeKey = "JWT"

func (c *Client) Middleware(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		neededScopes, required := jwtScopes(ctx.Operation())
		if !required {
			next(ctx)

			return
		}

		token, sub, ok := verifiedToken(ctx, c.cache, c.cfg)
		if !ok {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		if !hasAllScopes(token, neededScopes) {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")

			return
		}

		user, err := c.db.GetUserBySub(ctx.Context(), sub)
		if errors.Is(err, pgx.ErrNoRows) {
			c.logger.Warn("verified subject is not provisioned", zap.String("sub", sub))
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")

			return
		}

		if err != nil {
			c.logger.Error(
				"error resolving user in middleware", zap.String("sub", sub), zap.Error(err),
			)
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal Server Error")

			return
		}

		next(huma.WithValue(ctx, userKey, user))
	}
}

func jwtScopes(operation *huma.Operation) ([]string, bool) {
	for _, requirement := range operation.Security {
		if scopes, ok := requirement[JWTSchemeKey]; ok {
			return scopes, true
		}
	}

	return nil, false
}

func verifiedToken(ctx huma.Context, cache *jwk.Cache, cfg Config) (jwt.Token, string, bool) {
	raw, ok := bearerToken(ctx.Header("Authorization"))
	if !ok {
		return nil, "", false
	}

	token, err := parseToken(ctx.Context(), cache, cfg, raw)
	if err != nil {
		return nil, "", false
	}

	sub, ok := token.Subject()
	if !ok || sub == "" {
		return nil, "", false
	}

	return token, sub, true
}

// MustSingleJWTRequirement prevents multiple JWT-naming security requirements.
//
// This is seen as a configuration error from the developer.
func MustSingleJWTRequirement(_ *huma.OpenAPI, operation *huma.Operation) {
	count := 0

	for _, requirement := range operation.Security {
		if _, ok := requirement[JWTSchemeKey]; ok {
			count++
		}
	}

	if count > 1 {
		panic(fmt.Sprintf(
			"operation %q declares %d JWT security requirements; at most one is supported",
			operation.OperationID,
			count,
		))
	}
}

func parseToken(
	ctx context.Context,
	cache *jwk.Cache,
	cfg Config,
	token string,
) (jwt.Token, error) {
	keySet, err := cache.Lookup(ctx, cfg.JWKSURL)
	if err != nil {
		return nil, err
	}

	// A lot of providers omit the signing algorithm, but it can be inferred from the key type.

	return jwt.ParseString(token,
		jwt.WithKeySet(keySet, jws.WithInferAlgorithmFromKey(true)),
		jwt.WithValidate(true),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.Audience),
	)
}

func hasAllScopes(token jwt.Token, wanted []string) bool {
	granted := grantedScopes(token)

	for _, scope := range wanted {
		if !slices.Contains(granted, scope) {
			return false
		}
	}

	return true
}

func grantedScopes(token jwt.Token) []string {
	var scope string
	if err := token.Get("scope", &scope); err != nil {
		return nil
	}

	return strings.Fields(scope)
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "

	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}

	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}

	return token, true
}

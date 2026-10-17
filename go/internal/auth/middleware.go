package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/sharktype/satchel/generated"
	"go.uber.org/zap"
)

type contextKey int

const (
	subjectKey contextKey = iota
	userKey
)

func SubjectFromContext(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(subjectKey).(string)

	return s, ok && s != ""
}

func ViewerFromContext(ctx context.Context) (generated.User, bool) {
	user, ok := ctx.Value(userKey).(generated.User)

	return user, ok
}

// RequireUserMiddleware requires an authenticated user is in context.
//
// It must be registered after NewJWTAuthMiddleware, and does not test authorization; just that
// the user exists at all; i.e. is not anonymous.
func RequireUserMiddleware(
	api huma.API, manager *Manager, logger *zap.Logger,
) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		sub, ok := SubjectFromContext(ctx.Context())
		if !ok {
			next(ctx)

			return
		}

		user, err := manager.RequireUser(ctx.Context(), sub)
		if errors.Is(err, ErrUserNotProvisioned) {
			logger.Warn("verified subject is not provisioned", zap.String("sub", sub))
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")

			return
		}

		if err != nil {
			logger.Error("error resolving user in middleware", zap.Error(err))
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal Server Error")

			return
		}

		next(huma.WithValue(ctx, userKey, user))
	}
}

func NewJWTAuthMiddleware(
	api huma.API, cache *jwk.Cache, cfg Config,
) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		required := false

		var neededScopes []string
		for _, requirement := range ctx.Operation().Security {
			if scopes, ok := requirement[DefaultJWTSchemeKey]; ok {
				required = true
				neededScopes = scopes

				break
			}
		}

		if !required {
			next(ctx)

			return
		}

		token, ok := bearerToken(ctx.Header("Authorization"))
		if !ok {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		parsed, err := parseToken(ctx.Context(), cache, cfg, token)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		sub, ok := parsed.Subject()
		if !ok || sub == "" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		ctx = huma.WithValue(ctx, subjectKey, sub)

		if !hasAllScopes(parsed, neededScopes) {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")

			return
		}

		next(ctx)
	}
}

// MustSingleJWTRequirement prevents multiple JWT-naming security requirements.
//
// This is seen as a configuration error from the developer.
func MustSingleJWTRequirement(_ *huma.OpenAPI, operation *huma.Operation) {
	count := 0

	for _, requirement := range operation.Security {
		if _, ok := requirement[DefaultJWTSchemeKey]; ok {
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

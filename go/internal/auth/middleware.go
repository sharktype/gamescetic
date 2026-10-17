package auth

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"go.uber.org/zap"
)

type contextKey int

const (
	subjectKey contextKey = iota
	userKey
)

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
	api huma.API, cache *jwk.Cache, jwksURL, issuer, audience string,
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

		keySet, err := cache.Lookup(ctx.Context(), jwksURL)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		parsed, err := jwt.ParseString(token,
			jwt.WithKeySet(keySet),
			jwt.WithValidate(true),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(audience),
		)
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
	var claim any
	if err := token.Get("scopes", &claim); err != nil {
		return nil
	}

	switch scopes := claim.(type) {
	case []string:
		return scopes
	case []any:
		granted := make([]string, 0, len(scopes))

		for _, entry := range scopes {
			if scope, ok := entry.(string); ok {
				granted = append(granted, scope)
			}
		}

		return granted
	}

	return nil
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

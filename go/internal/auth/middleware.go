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

// NewJWTAuthMiddleware places the subject based on JWT auth in the context.
func NewJWTAuthMiddleware(
	api huma.API, cache *jwk.Cache, jwksURL, issuer, audience string,
) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		var neededScopes []string
		required := false

		for _, opScheme := range ctx.Operation().Security {
			if scopes, ok := opScheme[DefaultJWTSchemeKey]; ok {
				neededScopes = scopes
				required = true

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

		if len(neededScopes) == 0 {
			next(ctx)

			return
		}

		next(ctx)
	}
}

					return
				}
			}
		}

		_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")
	}
}

// bearerToken extracts the token from an Authorization header value.
func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "

	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}

	token := strings.TrimSpace(header[len(prefix):])
	if token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}

	return token, true
}

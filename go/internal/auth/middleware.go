package auth

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/sharktype/satchel/internal/clients"
	"go.uber.org/zap"
)

type contextKey int

const subjectKey contextKey = iota

// EnsureUserMiddleware must run after the JWT middleware; otherwise unverified token.
func EnsureUserMiddleware(
	clients *clients.Clients, logger *zap.Logger,
) func(http.Handler) http.Handler {
	manager := NewManager(clients.DB)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")

			if token != "" {
				if err := manager.EnsureUser(request.Context(), token); err != nil {
					logger.Error("error ensuring user in middleware", zap.Error(err))
					http.Error(writer, "Unauthorized", http.StatusUnauthorized)

					return
				}
			}

			next.ServeHTTP(writer, request)
		})
	}
}

func NewJWTAuthMiddleware(
	api huma.API, cfg *Config, cache *jwk.Cache,
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

		token := strings.TrimPrefix(ctx.Header("Authorization"), "Bearer ")
		if token == "" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		keySet, err := cache.Lookup(ctx.Context(), cfg.JWKSURL)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		parsed, err := jwt.ParseString(token,
			jwt.WithKeySet(keySet),
			jwt.WithValidate(true),
			jwt.WithIssuer(cfg.Issuer),
			jwt.WithAudience(cfg.Audience),
		)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		if len(neededScopes) == 0 {
			next(ctx)

			return
		}

		var scopes []string
		if err := parsed.Get("scopes", &scopes); err == nil {
			for _, scope := range scopes {
				if slices.Contains(neededScopes, scope) {
					next(ctx)

					return
				}
			}
		}

		_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")
	}
}

// SubjectMiddleware extracts the subject without re-verifying the token; middleware already did.
func SubjectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
		if token != "" {
			if parsed, err := jwt.ParseInsecure([]byte(token)); err == nil {
				if sub, ok := parsed.Subject(); ok && sub != "" {
					request = request.WithContext(context.WithValue(request.Context(), subjectKey, sub))
				}
			}
		}

		next.ServeHTTP(writer, request)
	})
}

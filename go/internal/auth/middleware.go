package auth

import (
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
		neededScopes, _, err := JWTRequirement(ctx.Operation())
		if err != nil {
			c.logger.Error("invalid JWT security requirement", zap.Error(err))
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal Server Error")

			return
		}

		raw, err := bearerToken(ctx.Header("Authorization"))
		if err != nil {
			c.logger.Debug("rejected token", zap.Error(err))
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		keySet, err := c.cache.Lookup(ctx.Context(), c.cfg.JWKSURL)
		if err != nil {
			c.logger.Error("could not load JWT signing keys", zap.Error(err))
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal Server Error")

			return
		}

		identity, err := verifiedToken(raw, keySet, c.cfg)
		if err != nil {
			c.logger.Debug("rejected token", zap.Error(err))
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "Unauthorized")

			return
		}

		if !hasAllScopes(identity.token, neededScopes) {
			c.logger.Debug("token lacks required scopes", zap.String("sub", identity.sub))
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")

			return
		}

		user, err := c.db.GetUserBySub(ctx.Context(), identity.sub)
		if errors.Is(err, pgx.ErrNoRows) {
			c.logger.Warn("verified subject is not provisioned", zap.String("sub", identity.sub))
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "Forbidden")

			return
		}

		if err != nil {
			c.logger.Error(
				"error resolving user in middleware", zap.String("sub", identity.sub), zap.Error(err),
			)
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "Internal Server Error")

			return
		}

		next(huma.WithValue(ctx, userKey, rowToUser(user)))
	}
}

// JWTRequirement returns the scopes of the operation's JWT security requirement.
//
// More than one JWT requirement is not supported.
func JWTRequirement(operation *huma.Operation) (scopes []string, declared bool, err error) {
	for _, requirement := range operation.Security {
		requirementScopes, ok := requirement[JWTSchemeKey]
		if !ok {
			continue
		}

		if declared {
			return nil, true, fmt.Errorf(
				"operation %q declares more than one JWT security requirement",
				operation.OperationID,
			)
		}

		scopes, declared = requirementScopes, true
	}

	return scopes, declared, nil
}

type principal struct {
	token jwt.Token
	sub   string
}

func verifiedToken(raw string, keySet jwk.Set, cfg Config) (principal, error) {
	token, err := jwt.ParseString(raw,
		jwt.WithKeySet(keySet, jws.WithInferAlgorithmFromKey(true)),
		jwt.WithValidate(true),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.Audience),
	)
	if err != nil {
		return principal{}, fmt.Errorf("invalid token: %w", err)
	}

	sub, ok := token.Subject()
	if !ok || sub == "" {
		return principal{}, errors.New("token has no subject")
	}

	return principal{token: token, sub: sub}, nil
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

func bearerToken(header string) (string, error) {
	const prefix = "Bearer "

	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", errors.New("authorization header is not a bearer token")
	}

	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", errors.New("bearer token is empty")
	}

	return token, nil
}

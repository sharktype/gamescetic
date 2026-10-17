package routing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/health"
	"github.com/sharktype/satchel/internal/utils"
	"go.uber.org/zap"
)

func MustHandle(
	ctx context.Context,
	logger *zap.Logger,
	r *chi.Mux,
	appClients *clients.Clients,
	title, version, jwksURL, issuer, audience string,
) {
	utils.Must(
		handle(ctx, logger, r, appClients, title, version, jwksURL, issuer, audience),
		"error setting up routing",
	)
}

func handle(
	ctx context.Context,
	logger *zap.Logger,
	r *chi.Mux,
	clients *clients.Clients,
	title, version, jwksURL, issuer, audience string,
) error {
	humaConfig := huma.DefaultConfig(title, version)
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		auth.DefaultJWTSchemeKey: {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	humaConfig.OnAddOperation = append(humaConfig.OnAddOperation, auth.MustSingleJWTRequirement)

	if jwksURL == "" || jwksURL == "<snip>" {
		return fmt.Errorf("AUTH_JWKSURL is not set in the environment variables")
	}

	jwtCache, err := auth.NewJWKSet(ctx, jwksURL)
	if err != nil {
		return err
	}

	// huma registers these as plain routes; gate on the router rather than through the middleware.

	r.Use(auth.RequireTokenForPaths(
		jwtCache,
		jwksURL,
		issuer,
		audience,
		[]string{
			humaConfig.DocsPath,
			humaConfig.OpenAPIPath + ".json",
			humaConfig.OpenAPIPath + ".yaml",
			humaConfig.OpenAPIPath + "-3.0.json",
			humaConfig.OpenAPIPath + "-3.0.yaml",
		},
	))

	// Reminder: default body size limit is 1 MiB.

	// Group 1 (rate limit 50 /s). Also, remember global rate limiting is 100 /s (middleware).

	r.Group(func(r chi.Router) {
		r.Use(httprate.Limit(
			50,
			rateLimitWindow,
			httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		))

		api := MustNewAPI(r, humaConfig, jwtCache, jwksURL, issuer, audience, clients, logger)

		huma.Register(api, huma.Operation{
			OperationID: "GetHealth",
			Summary:     "Get Health",
			Method:      http.MethodGet,
			Path:        "/health",
		}, withErrorLogging(logger, "GetHealth", health.GetHealth))
	})

	// Group 2 (rate limit 30 /s).

	r.Group(func(r chi.Router) {
		r.Use(httprate.Limit(
			30,
			rateLimitWindow,
			httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		))

		api := MustNewAPI(r, humaConfig, jwtCache, jwksURL, issuer, audience, clients, logger)

		if clients.DB != nil {
			huma.Register(
				api,
				huma.Operation{
					OperationID: "GetDatabaseHealth",
					Summary:     "Get Database Health",
					Method:      http.MethodGet,
					Path:        "/health/db",
					Security: []map[string][]string{
						{"JWT": {}},
					},
				},
				withErrorLogging(
					logger, "GetDatabaseHealth", health.GetDatabaseHealth(logger, clients),
				),
			)
		}

		huma.Register(
			api,
			huma.Operation{
				OperationID: "GetAuthHealth",
				Summary:     "Get Auth Health",
				Method:      http.MethodGet,
				Path:        "/health/auth",
				Security: []map[string][]string{
					{"JWT": {}},
				},
			},
			withErrorLogging(logger, "GetAuthHealth", health.GetAuthHealth),
		)

		huma.Register(
			api,
			huma.Operation{
				OperationID: "GetErrorHealth",
				Summary:     "Get Error Health",
				Method:      http.MethodGet,
				Path:        "/health/error",
				Security: []map[string][]string{
					{"JWT": {}},
				},
			},
			withErrorLogging(logger, "GetErrorHealth", health.GetErrorHealth),
		)
	})

	return nil
}

func withErrorLogging[I, O any](
	logger *zap.Logger,
	operation string,
	callback func(context.Context, *I) (*O, error),
) func(context.Context, *I) (*O, error) {
	// TODO: Report these to a bug tracker (Sentry or similar).

	return func(ctx context.Context, input *I) (*O, error) {
		result, err := callback(ctx, input)
		if err != nil {
			logger.Error("router returned error", zap.String("operation", operation), zap.Error(err))
		}

		return result, err
	}
}

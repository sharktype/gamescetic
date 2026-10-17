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
	humaConfig huma.Config,
	authConfig auth.Config,
) {
	utils.Must(
		handle(ctx, logger, r, appClients, humaConfig, authConfig),
		"error setting up routing",
	)
}

func handle(
	ctx context.Context,
	logger *zap.Logger,
	r *chi.Mux,
	clients *clients.Clients,
	humaConfig huma.Config,
	authConfig auth.Config,
) error {
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		auth.DefaultJWTSchemeKey: {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	humaConfig.OnAddOperation = append(humaConfig.OnAddOperation, auth.MustSingleJWTRequirement)

	if authConfig.JWKSURL == "" {
		return fmt.Errorf("AUTH_JWKSURL is not set in the environment variables")
	}

	jwtCache, err := auth.NewJWKSet(ctx, authConfig.JWKSURL)
	if err != nil {
		return err
	}

	r.Group(func(r chi.Router) {
		healthOp := huma.Operation{
			OperationID: "GetHealth",
			Summary:     "Get Health",
			Method:      http.MethodGet,
			Path:        "/health",
		}

		databaseOp := huma.Operation{
			OperationID: "GetDatabaseHealth",
			Summary:     "Get Database Health",
			Method:      http.MethodGet,
			Path:        "/health/db",
			Security: []map[string][]string{
				{auth.DefaultJWTSchemeKey: {}},
			},
		}

		authOp := huma.Operation{
			OperationID: "GetAuthHealth",
			Summary:     "Get Auth Health",
			Method:      http.MethodGet,
			Path:        "/health/auth",
			Security: []map[string][]string{
				{auth.DefaultJWTSchemeKey: {}},
			},
		}

		errorOp := huma.Operation{
			OperationID: "GetErrorHealth",
			Summary:     "Get Error Health",
			Method:      http.MethodGet,
			Path:        "/health/error",
			Security: []map[string][]string{
				{auth.DefaultJWTSchemeKey: {}},
			},
		}

		r.Use(limitByRoute(map[string]func(http.Handler) http.Handler{
			healthOp.Path:   httprate.Limit(publicRateLimit, rateLimitWindow),
			databaseOp.Path: httprate.Limit(securedRateLimit, rateLimitWindow),
			authOp.Path:     httprate.Limit(securedRateLimit, rateLimitWindow),
			errorOp.Path:    httprate.Limit(securedRateLimit, rateLimitWindow),
		}))

		api := MustNewAPI(r, humaConfig, jwtCache, authConfig, clients, logger)

		huma.Register(api, healthOp, withErrorLogging(logger, healthOp, health.GetHealth))
		huma.Register(
			api,
			databaseOp,
			withErrorLogging(logger, databaseOp, health.GetDatabaseHealth(logger, clients)),
		)
		huma.Register(api, authOp, withErrorLogging(logger, authOp, health.GetAuthHealth))
		huma.Register(api, errorOp, withErrorLogging(logger, errorOp, health.GetErrorHealth))
	})

	return nil
}

func withErrorLogging[I, O any](
	logger *zap.Logger,
	operation huma.Operation,
	callback func(context.Context, *I) (*O, error),
) func(context.Context, *I) (*O, error) {
	// TODO: Report these to a bug tracker (Sentry or similar).

	return func(ctx context.Context, input *I) (*O, error) {
		result, err := callback(ctx, input)
		if err != nil {
			logger.Error(
				"router returned error",
				zap.String("operation", operation.OperationID),
				zap.Error(err),
			)
		}

		return result, err
	}
}

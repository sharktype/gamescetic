package routing

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/health"
	"go.uber.org/zap"
)

func Handle(
	ctx context.Context,
	logger *zap.Logger,
	r *chi.Mux,
	clients *clients.Clients,
	humaConfig huma.Config,
	authConfig auth.Config,
) error {
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		auth.JWTSchemeKey: {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	humaConfig.OnAddOperation = append(humaConfig.OnAddOperation, auth.MustSingleJWTRequirement)

	jwtCache, err := auth.NewJWKSet(ctx, authConfig.JWKSURL)
	if err != nil {
		return err
	}

	public, secured := NewAPI(r, humaConfig, jwtCache, authConfig, clients, logger)

	register(logger, public, huma.Operation{
		OperationID: "GetHealth",
		Summary:     "Get Health",
		Method:      http.MethodGet,
		Path:        "/health",
	}, health.GetHealth)

	register(logger, secured, huma.Operation{
		OperationID: "GetDatabaseHealth",
		Summary:     "Get Database Health",
		Method:      http.MethodGet,
		Path:        "/health/db",
	}, health.GetDatabaseHealth(logger, clients))

	register(logger, secured, huma.Operation{
		OperationID: "GetAuthHealth",
		Summary:     "Get Auth Health",
		Method:      http.MethodGet,
		Path:        "/health/auth",
	}, health.GetAuthHealth)

	register(logger, secured, huma.Operation{
		OperationID: "GetErrorHealth",
		Summary:     "Get Error Health",
		Method:      http.MethodGet,
		Path:        "/health/error",
	}, health.GetErrorHealth)

	return nil
}

func register[I, O any](
	logger *zap.Logger,
	api huma.API,
	operation huma.Operation,
	callback func(context.Context, *I) (*O, error),
) {
	// TODO: Report these to a bug tracker (Sentry or similar).

	huma.Register(api, operation, func(ctx context.Context, input *I) (*O, error) {
		result, err := callback(ctx, input)
		if err != nil {
			logger.Error(
				"router returned error",
				zap.String("operation", operation.OperationID),
				zap.Error(err),
			)
		}

		return result, err
	})
}

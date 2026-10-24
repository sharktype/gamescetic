package routing

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sharktype/satchel/internal/bugtracker"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/health"
	"github.com/sharktype/satchel/internal/logger"
	"go.uber.org/zap"
)

func Handle(
	mux *chi.Mux,
	appClients *clients.Clients,
	humaConfig huma.Config,
) {
	public, secured := newAPI(mux, humaConfig, appClients.Auth)

	register(public, huma.Operation{
		OperationID: "GetHealth",
		Summary:     "Get Health",
		Method:      http.MethodGet,
		Path:        "/health",
	}, health.GetHealth)

	register(secured, huma.Operation{
		OperationID: "GetDatabaseHealth",
		Summary:     "Get Database Health",
		Method:      http.MethodGet,
		Path:        "/health/db",
	}, health.GetDatabaseHealth(appClients.DB))

	register(secured, huma.Operation{
		OperationID: "GetAuthHealth",
		Summary:     "Get Auth Health",
		Method:      http.MethodGet,
		Path:        "/health/auth",
	}, health.GetAuthHealth)

	register(secured, huma.Operation{
		OperationID: "GetErrorHealth",
		Summary:     "Get Error Health",
		Method:      http.MethodGet,
		Path:        "/health/error",
	}, health.GetErrorHealth)
}

func register[I, O any](
	api huma.API,
	operation huma.Operation,
	callback func(context.Context, *I) (*O, error),
) {
	huma.Register(api, operation, func(ctx context.Context, input *I) (*O, error) {
		result, err := callback(ctx, input)
		if err == nil {
			return result, nil
		}

		var statusErr huma.StatusError
		if !errors.As(err, &statusErr) {
			// Huma turns this into a 500.

			logger.Error(
				"handler failed",
				zap.String("operation", operation.OperationID),
				zap.Error(err),
			)

			bugtracker.Notify(err)

			return result, err
		}

		if statusErr.GetStatus() < http.StatusInternalServerError {
			return result, err
		}

		logErr := err
		if cause := errors.Unwrap(statusErr); cause != nil {
			logErr = cause
		}

		logger.Error(
			"handler failed",
			zap.String("operation", operation.OperationID),
			zap.Int("status", statusErr.GetStatus()),
			zap.Error(logErr),
		)

		bugtracker.Notify(logErr)

		return result, err
	})
}

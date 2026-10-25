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

	register(public, huma.Operation{
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

		// Huma returns a 500 for any error that is not a status error.

		status := http.StatusInternalServerError
		reportErr := err

		if statusErr, ok := errors.AsType[huma.StatusError](err); ok {
			status = statusErr.GetStatus()
			if status < http.StatusInternalServerError {
				return result, err
			}

			if cause := errors.Unwrap(statusErr); cause != nil {
				reportErr = cause
			}
		}

		bugtracker.ReportError(
			ctx,
			"handler failed",
			reportErr,
			zap.String("operation", operation.OperationID),
			zap.Int("status", status),
		)

		return result, err
	})
}

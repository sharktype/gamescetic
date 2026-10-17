package health

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"go.uber.org/zap"
)

type GetHealthInput struct{}

type GetHealthOutput struct {
	Body struct {
		Message string `json:"message" example:"OK" doc:"health check response message"`
	}
}

type GetAuthHealthOutput struct {
	Body struct {
		Message string `json:"message" example:"user-1 OK" doc:"health check response message"`
	}
}

func GetHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	resp := &GetHealthOutput{}
	resp.Body.Message = "OK"

	return resp, nil
}

func GetAuthHealth(ctx context.Context, _ *GetHealthInput) (*GetAuthHealthOutput, error) {
	resp := &GetAuthHealthOutput{}

	user, ok := auth.ViewerFromContext(ctx)
	if !ok {
		resp.Body.Message = "no authenticated user in context"

		return resp, huma.Error500InternalServerError("authenticated request has no user")
	}

	resp.Body.Message = fmt.Sprintf("%s OK", user.ID)

	return resp, nil
}

func GetDatabaseHealth(
	logger *zap.Logger, clients *clients.Clients,
) func(context.Context, *GetHealthInput) (*GetHealthOutput, error) {
	return func(ctx context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
		resp := &GetHealthOutput{}

		if _, err := clients.DB.Placeholder(ctx); err != nil {
			logger.Error("database health check failed", zap.Error(err))

			return resp, huma.Error503ServiceUnavailable("database unavailable")
		}

		resp.Body.Message = "database connection OK"

		return resp, nil
	}
}

func GetErrorHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	return nil, huma.Error500InternalServerError("intentional error for health check")
}

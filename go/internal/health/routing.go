package health

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/clients"
	"go.uber.org/zap"
)

type GetHealthInput struct{}

type GetHealthOutput struct {
	Body struct {
		Message string `json:"message" example:"OK" doc:"health check response message"`
	}
}

func GetHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	resp := &GetHealthOutput{}
	resp.Body.Message = "OK"

	return resp, nil
}

func GetAuthHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	resp := &GetHealthOutput{}
	resp.Body.Message = "OK"

	return resp, nil
}

func GetDatabaseHealth(
	logger *zap.Logger, clients *clients.Clients,
) func(context.Context, *GetHealthInput) (*GetHealthOutput, error) {
	return func(ctx context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
		resp := &GetHealthOutput{}
		_, err := clients.DB.Placeholder(ctx)
		if err != nil {
			logger.Error("database health check failed")

			resp.Body.Message = "database connection failed"

			return resp, huma.Error503ServiceUnavailable("database unavailable")
		}

		resp.Body.Message = "database connection OK"

		return resp, nil
	}
}

func GetErrorHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	resp := &GetHealthOutput{}
	resp.Body.Message = "error health check"

	return resp, huma.Error500InternalServerError("intentional error for health check")
}

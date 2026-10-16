package health

import (
	"context"
	"fmt"

	"github.com/sharktype/satchel/internal/clients"
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

func GetDatabaseHealth(clients *clients.Clients) func(context.Context, *GetHealthInput) (*GetHealthOutput, error) {
	return func(ctx context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
		resp := &GetHealthOutput{}
		_, err := clients.DB.Placeholder(ctx)
		if err != nil {
			resp.Body.Message = "database connection failed"

			return resp, err
		}
		resp.Body.Message = "database connection OK"

		return resp, nil
	}
}

func GetErrorHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	resp := &GetHealthOutput{}
	resp.Body.Message = "error health check"

	return resp, fmt.Errorf("intentional error for health check")
}

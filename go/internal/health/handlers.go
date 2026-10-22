package health

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/httpx"
)

type GetHealthInput struct{}

type GetHealthOutput struct {
	Body struct {
		Message string `json:"message" example:"OK" doc:"health check response message"`
	}
}

type GetAuthHealthOutput struct {
	Body struct {
		Message string `json:"message" example:"user-1 (Ada, ada-lovelace@example.com) OK" doc:"health check response message"`
	}
}

func GetHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	resp := &GetHealthOutput{}
	resp.Body.Message = "OK"

	return resp, nil
}

func GetAuthHealth(ctx context.Context, _ *GetHealthInput) (*GetAuthHealthOutput, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		// Defensive: this actually isn't expected to be reachable since the middleware should 403
		// when the user is not authenticated.

		return nil, huma.Error500InternalServerError("authenticated request has no user")
	}

	resp := &GetAuthHealthOutput{}
	resp.Body.Message = fmt.Sprintf("%s (%s, %s) OK", user.ID, user.Name, user.Email)

	return resp, nil
}

func GetDatabaseHealth(
	database *db.Store,
) func(context.Context, *GetHealthInput) (*GetHealthOutput, error) {
	return func(ctx context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
		resp := &GetHealthOutput{}

		if err := database.Ping(ctx); err != nil {
			return nil, httpx.NewStatusError(
				http.StatusServiceUnavailable, "database unavailable", err,
			)
		}

		resp.Body.Message = "database connection OK"

		return resp, nil
	}
}

func GetErrorHealth(_ context.Context, _ *GetHealthInput) (*GetHealthOutput, error) {
	return nil, huma.Error500InternalServerError("intentional error for health check")
}

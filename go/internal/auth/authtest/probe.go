package authtest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/auth"
)

const ProbePath = "/probe"

type probeInput struct{}

type probeOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type Probe struct {
	Ran  bool
	User generated.User
}

func RegisterProbe(api huma.API, security []map[string][]string) *Probe {
	recorded := &Probe{}

	huma.Register(api, huma.Operation{
		OperationID: "Probe",
		Method:      http.MethodGet,
		Path:        ProbePath,
		Security:    security,
	}, func(ctx context.Context, _ *probeInput) (*probeOutput, error) {
		recorded.Ran = true
		recorded.User, _ = auth.UserFromContext(ctx)

		return &probeOutput{}, nil
	})

	return recorded
}

func RequireJWT(scopes ...string) []map[string][]string {
	return []map[string][]string{{auth.JWTSchemeKey: scopes}}
}

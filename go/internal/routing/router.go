package routing

import (
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

const (
	RequestTimeout   = 60 * time.Second
	publicRateLimit  = 30
	securedRateLimit = 60
	compressionLevel = 5
)

// newAPI returns the groups on which every operation must be registered as a n-ary tuple.
func newAPI(
	r chi.Router, humaConfig huma.Config, authClient *auth.Client,
) (public *huma.Group, secured *huma.Group) {
	schemes := make(
		map[string]*huma.SecurityScheme, len(humaConfig.Components.SecuritySchemes)+1,
	)

	maps.Copy(schemes, humaConfig.Components.SecuritySchemes)
	schemes[auth.JWTSchemeKey] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}
	humaConfig.Components.SecuritySchemes = schemes

	api := humachi.New(r, humaConfig)

	public = huma.NewGroup(api)
	public.UseSimpleModifier(mustNotDeclareJWT)
	public.UseMiddleware(rateLimit(publicRateLimit))

	secured = huma.NewGroup(api)
	secured.UseSimpleModifier(declareJWT)
	secured.UseMiddleware(rateLimit(securedRateLimit), authClient.Middleware(api))

	return public, secured
}

// NewRouter returns the transport-level handler.
//
// Every request passes through this router including ones huma/the API never sees such as /ping.
//
// Middleware here apply to _all_ traffic, and middleware in the API groups are layered on top.
func NewRouter(logger *zap.Logger, corsMiddleware func(http.Handler) http.Handler) *chi.Mux {
	router := chi.NewMux()

	router.Use(httpx.SecurityHeaders())
	router.Use(middleware.RequestID)
	router.Use(clientIP)
	router.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger: zap.NewStdLog(logger),
	}))
	router.Use(middleware.Recoverer)
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.GetHead)
	router.Use(middleware.Timeout(RequestTimeout))
	router.Use(middleware.Compress(compressionLevel))
	router.Use(corsMiddleware)

	// Counts per replica so remember to keep volumetric throttling at the edge too.

	router.Use(rateLimitGlobally)
	router.Use(middleware.Heartbeat("/ping"))

	return router
}

// declareJWT marks the given operation as requiring a JWT token.
//
// If the operation already declares a JWT requirement, e.g. to ask for scopes, it is left as is.
// This also matters because Huma calls declareJWT twice per operation: once when adding it to
// the OpenAPI spec, so the spec shows that a JWT is needed, and once when connecting it to the
// router.
//
// If you're confused, Huma uses each operation for two things: describing it in the OpenAPI spec,
// and routing requests to it. Anything a group changes about an operation, such as a "/v1" path
// prefix or this JWT requirement, must apply to both, or the spec would not match how the server
// behaves. Huma cannot tell which changes matter to which, so it runs declareJWT for both.
func declareJWT(operation *huma.Operation) {
	for _, requirement := range operation.Security {
		if len(requirement) != 1 {
			panic(fmt.Sprintf(
				"operation %q must declare exactly one JWT security requirement, got %v",
				operation.OperationID, requirement,
			))
		}

		if _, ok := requirement[auth.JWTSchemeKey]; !ok {
			panic(fmt.Sprintf(
				"operation %q declares a non-JWT security requirement but is registered on the secured group",
				operation.OperationID,
			))
		}
	}

	_, declared, err := auth.JWTRequirement(operation)
	if err != nil {
		panic(err)
	}

	if !declared {
		operation.Security = append(
			operation.Security, map[string][]string{auth.JWTSchemeKey: {}},
		)
	}
}

func mustNotDeclareJWT(operation *huma.Operation) {
	_, declared, err := auth.JWTRequirement(operation)
	if err != nil {
		panic(err)
	}

	if declared {
		panic(fmt.Sprintf(
			"operation %q declares a JWT requirement but is registered on the public group",
			operation.OperationID,
		))
	}
}

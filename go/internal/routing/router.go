package routing

import (
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/httpx"
)

const (
	requestTimeout   = 60 * time.Second
	publicRateLimit  = 30
	securedRateLimit = 60
	compressionLevel = 5
)

// NewAPI returns the groups on which every operation must be registered.
func NewAPI(
	r chi.Router, humaConfig huma.Config, authClient *auth.Client,
) (public *huma.Group, secured *huma.Group) {
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		auth.JWTSchemeKey: {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

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
// Every request passes through this router, including ones huma/the API never sees, such as /ping.
//
// Middleware here apply to _all_ traffic, and middleware in the API groups are layered on top.
func NewRouter(corsMiddleware httpx.Middleware) *chi.Mux {
	router := chi.NewMux()

	// Assumption: API sits behind a proxy e.g. Cloudflare that sets forwarding headers and is not
	// directly reachable. Otherwise, RealIP would trust the given client-controlled IP address.
	//
	// This is fine for self-hosted configurations most of the time since either the API would be
	// deployed via a reverse proxy with TLS, or the API is only used by trusted clients.

	router.Use(httpx.SecurityHeaders())
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.GetHead)
	router.Use(middleware.Timeout(requestTimeout))
	router.Use(middleware.Compress(compressionLevel))
	router.Use(corsMiddleware)

	// Counts per replica so remember to keep volumetric throttling at the edge too.

	router.Use(rateLimitGlobally)
	router.Use(middleware.Heartbeat("/ping"))

	return router
}

// declareJWT marks every operation on the secured group as requiring a JWT token.
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
	requirements := auth.JWTRequirements(operation)

	if len(requirements) > 1 {
		panic(fmt.Sprintf(
			"operation %q declares %d JWT security requirements; at most one is supported",
			operation.OperationID,
			len(requirements),
		))
	}

	if len(requirements) == 0 {
		operation.Security = append(
			operation.Security, map[string][]string{auth.JWTSchemeKey: {}},
		)
	}
}

func mustNotDeclareJWT(operation *huma.Operation) {
	if len(auth.JWTRequirements(operation)) > 0 {
		panic(fmt.Sprintf(
			"operation %q declares a JWT requirement but is registered on the public group",
			operation.OperationID,
		))
	}
}

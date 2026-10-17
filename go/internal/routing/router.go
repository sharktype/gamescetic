package routing

import (
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

const (
	requestTimeout   = 60 * time.Second
	compressionLevel = 5

	globalRateLimit  = 120
	publicRateLimit  = 30
	securedRateLimit = 60
	rateLimitWindow  = time.Minute
)

// NewAPI returns the groups every operation must be registered on.
//
// Rate limiting must run before the JWT check such that requests with an invalid token still
// count towards the limit. Huma runs anything attached to the API before anything attached to a
// group, so the rate limiter and the JWT check are both attached to each group, in that order.
func NewAPI(
	r chi.Router,
	humaConfig huma.Config,
	jwtCache *jwk.Cache,
	authConfig auth.Config,
	dbClient db.Client,
	logger *zap.Logger,
) (public *huma.Group, secured *huma.Group) {
	api := humachi.New(r, humaConfig)

	public = huma.NewGroup(api)
	public.UseSimpleModifier(mustNotDeclareJWT)
	public.UseMiddleware(rateLimit(publicRateLimit))

	secured = huma.NewGroup(api)
	secured.UseSimpleModifier(declareJWT)
	secured.UseMiddleware(
		rateLimit(securedRateLimit),
		auth.NewJWTAuthMiddleware(api, jwtCache, authConfig, dbClient, logger),
	)

	return public, secured
}

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

	router.Use(httprate.LimitByIP(globalRateLimit, rateLimitWindow))
	router.Use(middleware.Heartbeat("/ping"))

	return router
}

func rateLimit(limit int) func(huma.Context, func(huma.Context)) {
	limiter := httprate.NewRateLimiter(limit, rateLimitWindow)

	return func(ctx huma.Context, next func(huma.Context)) {
		request, writer := humachi.Unwrap(ctx)

		ip, _ := httprate.KeyByIP(request)
		if limiter.RespondOnLimit(writer, request, ip+" "+ctx.Operation().OperationID) {
			return
		}

		next(ctx)
	}
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
	if !declaresJWT(operation) {
		operation.Security = append(
			operation.Security, map[string][]string{auth.JWTSchemeKey: {}},
		)
	}
}

func mustNotDeclareJWT(operation *huma.Operation) {
	if declaresJWT(operation) {
		panic(fmt.Sprintf(
			"operation %q declares a JWT requirement but is registered on the public group",
			operation.OperationID,
		))
	}
}

func declaresJWT(operation *huma.Operation) bool {
	for _, requirement := range operation.Security {
		if _, ok := requirement[auth.JWTSchemeKey]; ok {
			return true
		}
	}

	return false
}

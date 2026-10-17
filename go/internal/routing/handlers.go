package routing

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
)

const (
	requestTimeout   = 60 * time.Second
	compressionLevel = 5

	globalRateLimit = 100
	rateLimitWindow = time.Minute
)

func MustNewAPI(
	r chi.Router,
	humaConfig huma.Config,
	jwtCache *jwk.Cache,
	authConfig auth.Config,
	clients *clients.Clients,
	logger *zap.Logger,
) huma.API {
	api := humachi.New(r, humaConfig)

	api.UseMiddleware(auth.NewJWTAuthMiddleware(api, jwtCache, authConfig))
	api.UseMiddleware(auth.RequireUserMiddleware(api, auth.NewManager(clients.DB), logger))

	return api
}

func MustNewRouter(corsMiddleware httpx.Middleware) *chi.Mux {
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

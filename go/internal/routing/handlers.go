package routing

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/http"
	"go.uber.org/zap"
)

const (
	requestTimeout   = 60 * time.Second
	compressionLevel = 5
)

func MustNewAPI(
	r chi.Router,
	humaConfig huma.Config,
	jwtCache *jwk.Cache,
	jwksURL, issuer, audience string,
	clients *clients.Clients,
	logger *zap.Logger,
) huma.API {
	api := humachi.New(r, humaConfig)

	api.UseMiddleware(auth.NewJWTAuthMiddleware(api, jwtCache, jwksURL, issuer, audience))
	api.UseMiddleware(auth.RequireUserMiddleware(api, auth.NewManager(clients.DB), logger))

	return api
}

func MustNewRouter(
	allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders []string,
	allowCredentials bool,
	maxAge int,
) *chi.Mux {
	router := chi.NewMux()

	// Careful: we don't throttle; be careful and set up infra-level throttling/rate limiting.
	// TODO: Can we throttle?

	// We don't use middleware.RealIP since it rewrites RemoteAddr from the client and can be
	// attacker-controlled to bypass rate limits or other IP restrictions.

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.GetHead)
	router.Use(middleware.Timeout(requestTimeout))
	router.Use(middleware.Compress(compressionLevel))
	router.Use(middleware.Heartbeat("/ping"))
	router.Use(http.MustNewCORSHandler(
		allowedOrigins,
		allowedMethods,
		allowedHeaders,
		exposedHeaders,
		allowCredentials,
		maxAge,
	))

	return router
}

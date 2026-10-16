package routing

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/cors"
)

func NewAPI(
	r chi.Router, humaConfig huma.Config, jwtConfig *auth.Config, jwtCache *jwk.Cache,
) huma.API {
	api := humachi.New(r, humaConfig)

	api.UseMiddleware(auth.NewJWTAuthMiddleware(api, jwtConfig, jwtCache))

	return api
}

func NewRouter(config cors.Config) *chi.Mux {
	router := chi.NewMux()

	// Careful: we don't throttle; be careful and set up infra-level throttling/rate limiting.

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.GetHead)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.Compress(5))
	router.Use(middleware.Heartbeat("/ping"))
	router.Use(cors.New(config))

	return router
}

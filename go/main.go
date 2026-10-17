package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/httpx"
	"github.com/sharktype/satchel/internal/logger"
	"github.com/sharktype/satchel/internal/routing"
)

const readHeaderTimeout = 5 * time.Second
const readTimeout = 30 * time.Second

// WriteTimeout has to outlast the router's own middleware.Timeout to ensure the response is fully
// written before the middleware times out.

const writeTimeout = 70 * time.Second
const idleTimeout = 120 * time.Second
const shutdownTimeout = 30 * time.Second

type Options struct {
	Port int `help:"port to listen on" short:"p" default:"8888"`
}

func main() {
	ctx := context.Background()

	logger := logger.MustNew()
	cfg := config.MustNew(logger)

	appClients := clients.MustNew(ctx, logger, db.Config{DSN: cfg.Clients.DB.DSN})

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		router := routing.MustNewRouter(httpx.MustNewCORSHandler(httpx.CORSConfig{
			AllowedOrigins:   cfg.CORS.AllowedOrigins,
			AllowedMethods:   cfg.CORS.AllowedMethods,
			AllowedHeaders:   cfg.CORS.AllowedHeaders,
			ExposedHeaders:   cfg.CORS.ExposedHeaders,
			AllowCredentials: cfg.CORS.AllowCredentials,
			MaxAge:           cfg.CORS.MaxAge,
		}))

		routing.MustHandle(
			ctx,
			logger,
			router,
			appClients,
			huma.DefaultConfig(cfg.API.Title, cfg.API.Version),
			auth.Config{
				JWKSURL:  cfg.Auth.JWKSURL,
				Issuer:   cfg.Auth.Issuer,
				Audience: cfg.Auth.Audience,
			},
		)

		server := &http.Server{
			Addr:              fmt.Sprintf(":%d", options.Port),
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		}

		hooks.OnStart(func() {
			logger.Info(fmt.Sprintf("starting server on port %d", options.Port))
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		})

		hooks.OnStop(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error(fmt.Sprintf("error shutting down server: %v", err))
			}
		})
	})

	cli.Run()
}

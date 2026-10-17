package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/logger"
	"github.com/sharktype/satchel/internal/routing"
)

const readHeaderTimeout = 5 * time.Second
const readTimeout = 30 * time.Second

// WriteTimeout has to outlast the router's own middleware.Timeout to ensure the response is fully
// written before the middleware times out.

const writeHeaderTimeout = 10 * time.Second
const idleTimeout = 120 * time.Second

type Options struct {
	Port int `help:"port to listen on" short:"p" default:"8888"`
}

func main() {
	ctx := context.Background()

	logger, err := logger.New()
	if err != nil {
		panic(fmt.Sprintf("cannot start server; error creating logger: %v", err))
	}

	cfg, err := config.New(logger)
	if err != nil {
		panic(fmt.Sprintf("cannot start server; error loading config: %v", err))
	}

	appClients, err := clients.New(ctx, &cfg.Clients, logger)
	if err != nil {
		panic(fmt.Sprintf("cannot start server; error creating clients: %v", err))
	}

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		router, err := routing.NewRouter(cfg.CORS)
		if err != nil {
			panic(fmt.Sprintf("error setting up routing: %v", err))
		}

		err = routing.Handle(
			ctx,
			router,
			huma.DefaultConfig(cfg.API.Title, cfg.API.Version),
			cfg,
			appClients,
			logger,
		)
		if err != nil {
			panic(fmt.Sprintf("error setting up routing: %v", err))
		}

		server := &http.Server{
			Addr:              fmt.Sprintf(":%d", options.Port),
			Handler:           router,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeHeaderTimeout,
			IdleTimeout:       idleTimeout,
		}

		hooks.OnStart(func() {
			logger.Info(fmt.Sprintf("starting server on port %d", options.Port))
			if err := server.ListenAndServe(); err != nil {
				panic(err)
			}
		})
	})

	cli.Run()
}

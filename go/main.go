package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/logger"
	"github.com/sharktype/satchel/internal/routing"
)

// Note: You will probably not need to edit this file when copying this template.

// Options for the CLI. Pass `--port` or set the `SERVICE_PORT` env var.
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
		router := routing.NewRouter(cfg.CORS)

		err = routing.Handle(ctx, router, huma.DefaultConfig(cfg.API.Title, cfg.API.Version), cfg, appClients, logger)
		if err != nil {
			// This error can only happen when initializing the routing, not for runtime errors.

			panic(fmt.Sprintf("error setting up routing: %v", err))
		}

		hooks.OnStart(func() {
			logger.Info(fmt.Sprintf("starting server on port %d", options.Port))
			err := http.ListenAndServe(fmt.Sprintf(":%d", options.Port), router)
			if err != nil {
				// This will normally only happen if the server fails to start.

				panic(err)
			}
		})
	})

	cli.Run()
}

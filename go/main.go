package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/httpx"
	"github.com/sharktype/satchel/internal/logger"
	"github.com/sharktype/satchel/internal/routing"
	"go.uber.org/zap"
)

const readHeaderTimeout = 5 * time.Second
const readTimeout = 30 * time.Second

// WriteTimeout has to outlast the router's own middleware.Timeout to ensure the response is fully
// written before the middleware times out.

const writeTimeout = 70 * time.Second
const idleTimeout = 120 * time.Second
const shutdownTimeout = 30 * time.Second

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Some background goroutines such as the JWT key cache are still needed to finish serving
	// requests in-flight when the server is told to stop. As a result, we use appCtx to manage
	// them and only cancel it after all requests are done.

	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	// Load config and logger:

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	logger, err := logger.New(logger.Config{
		Production: cfg.Production,
		Verbose:    cfg.Verbose,
	})
	if err != nil {
		return fmt.Errorf("error creating logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	// Initialise app clients:

	if cfg.Production {
		logger.Info("app booting in production mode...")
	} else {
		logger.Info("app booting in development mode...")
	}

	appClients, err := clients.New(appCtx, logger, db.Config{DSN: cfg.Clients.DB.DSN})
	if err != nil {
		return fmt.Errorf("error creating clients: %w", err)
	}
	defer appClients.Close()

	// Initialise router:

	corsHandler, err := httpx.NewCORSHandler(httpx.CORSConfig{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		ExposedHeaders:   cfg.CORS.ExposedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	})
	if err != nil {
		return err
	}

	router := routing.NewRouter(corsHandler)

	if err := routing.Handle(
		appCtx,
		logger,
		router,
		appClients,
		huma.DefaultConfig(cfg.API.Title, cfg.API.Version),
		auth.Config{
			JWKSURL:  cfg.Auth.JWKSURL,
			Issuer:   cfg.Auth.Issuer,
			Audience: cfg.Auth.Audience,
		},
	); err != nil {
		return fmt.Errorf("error setting up routing: %w", err)
	}

	// Initialise server:

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Startup can be slow but we want to exit straight away on SIGINT. As a result, we only
	// capture SIGINT and SIGTERM from here once startup is done.
	//
	// Reminder: signalCtx.Done() is called on SIGINT or SIGTERM and still shuts down the server,
	// but gracefully.

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ListenAndServe blocks until the server stops, so it runs in the background and reports why
	// it stopped on serveErr.

	serveErr := make(chan error, 1)

	go func() {
		logger.Info("starting server", zap.Int("port", cfg.Port))
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	case <-signalCtx.Done():
	}

	// Stop capturing SIGINT so a second Ctrl-C during graceful shutdown exits straight away.

	stop()

	// Graceful shutdown:

	logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}

	return nil
}

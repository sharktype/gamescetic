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
	"github.com/sharktype/satchel/internal/bugtracker"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/httpx"
	internalLogger "github.com/sharktype/satchel/internal/logger"
	"github.com/sharktype/satchel/internal/routing"
	"go.uber.org/zap"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 120 * time.Second
	graceMargin       = 10 * time.Second
	writeTimeout      = routing.RequestTimeout + graceMargin
	shutdownTimeout   = routing.RequestTimeout + graceMargin
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (err error) {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	shutdownLogger, err := internalLogger.Init(internalLogger.Config{
		Production: cfg.Production,
		Verbose:    cfg.Verbose,
	})
	if err != nil {
		return fmt.Errorf("error creating logger: %w", err)
	}
	defer shutdownLogger()

	shutdownBugtracker, err := bugtracker.Init(cfg.Telemetry.SentryDSN, cfg.Production)
	if err != nil {
		return fmt.Errorf("error creating bugtracker: %w", err)
	}
	defer shutdownBugtracker()

	// We actually do want to capture startup errors in telemetry.

	defer func() {
		if err != nil {
			bugtracker.Report(err)
		}
	}()

	if cfg.Production {
		internalLogger.Info("app booting in production mode...")
	} else {
		internalLogger.Info("app booting in development mode...")
	}

	appClients, err := clients.New(
		context.Background(),
		db.Config{DSN: cfg.Clients.DB.DSN},
		auth.Config{
			JWKSURL:  cfg.Auth.JWKSURL,
			Issuer:   cfg.Auth.Issuer,
			Audience: cfg.Auth.Audience,
		},
	)
	if err != nil {
		return fmt.Errorf("error creating clients: %w", err)
	}
	defer appClients.Close()

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

	routing.Handle(
		router,
		appClients,
		huma.DefaultConfig(cfg.API.Title, cfg.API.Version),
	)

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
	// it stopped on receiving at serveErr.

	serveErr := make(chan error, 1)

	go func() {
		internalLogger.Info("starting server", zap.Int("port", cfg.Port))
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

	internalLogger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}

	return nil
}

package config

import (
	"slices"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("CLIENTS_DB_DSN", "postgres://localhost:5432/foobar")
	t.Setenv("AUTH_JWKSURL", "https://example.com/jwks")
	t.Setenv("AUTH_ISSUER", "https://example.com")
	t.Setenv("AUTH_AUDIENCE", "foobar")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://example.com,https://example.com")
	t.Setenv("CORS_ALLOWED_METHODS", "GET,POST")
	t.Setenv("CORS_ALLOWED_HEADERS", "Authorization,Content-Type")
	t.Setenv("CORS_EXPOSE_HEADERS", "Link")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	t.Setenv("CORS_MAX_AGE", "86400")
}

func TestNewConfig(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := newConfig(zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Clients.DB.DSN != "postgres://localhost:5432/foobar" {
		t.Fatalf("got DSN %q, want %q", cfg.Clients.DB.DSN, "postgres://localhost:5432/foobar")
	}

	if cfg.Auth.JWKSURL != "https://example.com/jwks" {
		t.Fatalf("got jwks url %q, want %q", cfg.Auth.JWKSURL, "https://example.com/jwks")
	}

	expectedOrigins := []string{"http://example.com", "https://example.com"}
	if !slices.Equal(cfg.CORS.AllowedOrigins, expectedOrigins) {
		t.Fatalf("got allowed origins %v, want %v", cfg.CORS.AllowedOrigins, expectedOrigins)
	}

	expectedMethods := []string{"GET", "POST"}
	if !slices.Equal(cfg.CORS.AllowedMethods, expectedMethods) {
		t.Fatalf("got allowed methods %v, want %v", cfg.CORS.AllowedMethods, expectedMethods)
	}

	expectedHeaders := []string{"Authorization", "Content-Type"}
	if !slices.Equal(cfg.CORS.AllowedHeaders, expectedHeaders) {
		t.Fatalf("got allowed headers %v, want %v", cfg.CORS.AllowedHeaders, expectedHeaders)
	}

	expectedExposed := []string{"Link"}
	if !slices.Equal(cfg.CORS.ExposedHeaders, expectedExposed) {
		t.Fatalf("got exposed headers %v, want %v", cfg.CORS.ExposedHeaders, expectedExposed)
	}

	if !cfg.CORS.AllowCredentials {
		t.Fatal("got allow credentials false, want true")
	}

	if cfg.CORS.MaxAge != 86400 {
		t.Fatalf("got max age %d, want %d", cfg.CORS.MaxAge, 86400)
	}
}

func TestNewConfigRequiresSettings(t *testing.T) {
	keys := []string{
		"CLIENTS_DB_DSN",
		"AUTH_JWKSURL",
		"AUTH_ISSUER",
		"AUTH_AUDIENCE",
		"CORS_ALLOWED_ORIGINS",
		"CORS_ALLOWED_METHODS",
		"CORS_ALLOWED_HEADERS",
		"CORS_EXPOSE_HEADERS",
		"CORS_ALLOW_CREDENTIALS",
		"CORS_MAX_AGE",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(key, "")

			_, err := newConfig(zap.NewNop())
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("got error %v, want it to mention %q", err, key)
			}
		})
	}
}

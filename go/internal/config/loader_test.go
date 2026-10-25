package config

import (
	"os"
	"slices"
	"strings"
	"testing"
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
	t.Run("every setting is read", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("VERBOSE", "true")
		t.Setenv("PORT", "8080")
		t.Setenv("TELEMETRY_SENTRY_DSN", "https://test@example.com/1")

		cfg, err := New()
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

		if cfg.Port != 8080 {
			t.Fatalf("got port %d, want %d", cfg.Port, 8080)
		}

		if !cfg.Verbose {
			t.Fatal("got verbose false, want true")
		}

		if cfg.Telemetry.SentryDSN != "https://test@example.com/1" {
			t.Fatalf(
				"got sentry DSN %q, want %q",
				cfg.Telemetry.SentryDSN,
				"https://test@example.com/1",
			)
		}
	})

	t.Run("the telemetry DSN is optional", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("TELEMETRY_SENTRY_DSN", "")

		cfg, err := New()
		if err != nil {
			t.Fatal(err)
		}

		if cfg.Telemetry.SentryDSN != "" {
			t.Fatalf("got sentry DSN %q, want none", cfg.Telemetry.SentryDSN)
		}
	})

	t.Run("every setting is required", func(t *testing.T) {
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

				_, err := New()
				if err == nil || !strings.Contains(err.Error(), key) {
					t.Fatalf("got error %v, want it to mention %q", err, key)
				}
			})
		}
	})
}

func TestIsProduction(t *testing.T) {
	t.Run("unset is not production", func(t *testing.T) {
		t.Setenv("PRODUCTION", "")

		if err := os.Unsetenv("PRODUCTION"); err != nil {
			t.Fatal(err)
		}

		production, err := IsProduction()
		if err != nil {
			t.Fatal(err)
		}

		if production {
			t.Fatal("got production")
		}
	})

	t.Run(`"1" is production`, func(t *testing.T) {
		t.Setenv("PRODUCTION", "1")

		production, err := IsProduction()
		if err != nil {
			t.Fatal(err)
		}

		if !production {
			t.Fatal("got not production")
		}
	})

	invalidTests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "zero", value: "0"},
		{name: "true", value: "true"},
		{name: "false", value: "false"},
	}

	for _, test := range invalidTests {
		t.Run("rejects "+test.name, func(t *testing.T) {
			t.Setenv("PRODUCTION", test.value)
			if _, err := IsProduction(); err == nil {
				t.Fatalf("expected an error for %q", test.value)
			}
		})
	}
}

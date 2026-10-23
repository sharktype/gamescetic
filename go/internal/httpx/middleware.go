package httpx

import (
	"errors"
	"fmt"
	nethttp "net/http"
	"strings"

	"github.com/go-chi/cors"
)

func NewCORSHandler(cfg CORSConfig) (func(nethttp.Handler) nethttp.Handler, error) {
	origins, err := normalize(cfg.AllowedOrigins)
	if err != nil {
		return nil, fmt.Errorf("invalid CORS configuration: %w", err)
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   cfg.AllowedMethods,
		AllowedHeaders:   cfg.AllowedHeaders,
		ExposedHeaders:   cfg.ExposedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}), nil
}

func SecurityHeaders() func(nethttp.Handler) nethttp.Handler {
	// Note this API will only ever serve plain HTTP behind a proxy or CDN. HSTS settings belong at
	// that layer, not here (e.g., Cloudflare's security settings).

	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, request *nethttp.Request) {
			writer.Header().Set("Content-Security-Policy", "default-src 'none'")
			writer.Header().Set("Referrer-Policy", "no-referrer")
			writer.Header().Set("X-Content-Type-Options", "nosniff")
			writer.Header().Set("X-Frame-Options", "DENY")

			next.ServeHTTP(writer, request)
		})
	}
}

func normalize(origins []string) ([]string, error) {
	if len(origins) == 0 {
		return nil, errors.New("AllowedOrigins is required; list at least one origin")
	}

	normalized := make([]string, len(origins))
	for i, origin := range origins {
		normalized[i] = strings.TrimSpace(origin)

		if normalized[i] == "" {
			return nil, errors.New("AllowedOrigins contains an empty origin")
		}

		if normalized[i] == "*" {
			return nil, errors.New(`AllowedOrigins must list explicit origins, not "*"`)
		}
	}

	return normalized, nil
}

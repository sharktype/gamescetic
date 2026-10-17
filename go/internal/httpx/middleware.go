package httpx

import (
	"errors"
	nethttp "net/http"
	"strings"

	"github.com/go-chi/cors"
	"github.com/sharktype/satchel/internal/utils"
)

type Middleware = func(next nethttp.Handler) nethttp.Handler

func MustNewCORSHandler(cfg CORSConfig) Middleware {
	utils.Must(validate(cfg.AllowedOrigins), "invalid CORS configuration")

	return cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   cfg.AllowedMethods,
		AllowedHeaders:   cfg.AllowedHeaders,
		ExposedHeaders:   cfg.ExposedHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}

func SecurityHeaders() Middleware {
	// Note this API will only ever serve plain HTTP behind a proxy or CDN, so HSTS belongs at that
	// hop, not here (e.g., Cloudflare's security settings).

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

func validate(origins []string) error {
	if len(origins) == 0 {
		return errors.New("CORS_ALLOWED_ORIGINS is required; list at least one origin")
	}

	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			return errors.New("CORS_ALLOWED_ORIGINS contains an empty origin")
		}

		if trimmed == "*" {
			return errors.New(`CORS_ALLOWED_ORIGINS must list explicit origins, not "*"`)
		}
	}

	return nil
}

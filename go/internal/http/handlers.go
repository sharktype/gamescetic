package http

import (
	"errors"
	nethttp "net/http"
	"strings"

	"github.com/go-chi/cors"
	"github.com/sharktype/satchel/internal/utils"
)

func MustNewCORSHandler(
	allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders []string,
	allowCredentials bool,
	maxAge int,
) func(next nethttp.Handler) nethttp.Handler {
	utils.Must(validate(allowedOrigins), "invalid CORS configuration")

	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		ExposedHeaders:   exposedHeaders,
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
	})
}

func SecurityHeaders() func(next nethttp.Handler) nethttp.Handler {
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

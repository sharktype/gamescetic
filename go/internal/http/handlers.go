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

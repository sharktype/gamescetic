package cors

import (
	"net/http"

	"github.com/go-chi/cors"
)

func New(config Config) (func(next http.Handler) http.Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   config.AllowedMethods,
		AllowedHeaders:   config.AllowedHeaders,
		ExposedHeaders:   config.ExposedHeaders,
		AllowCredentials: config.AllowCredentials,
		MaxAge:           config.MaxAge,
	}), nil
}

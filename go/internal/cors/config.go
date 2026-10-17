package cors

import (
	"errors"
	"strings"
)

type Config struct {
	AllowedOrigins   []string `env:"ALLOWED_ORIGINS"`
	AllowedMethods   []string `env:"ALLOWED_METHODS"`
	AllowedHeaders   []string `env:"ALLOWED_HEADERS"`
	ExposedHeaders   []string `env:"EXPOSED_HEADERS"`
	AllowCredentials bool     `env:"ALLOW_CREDENTIALS"`
	MaxAge           int      `env:"MAX_AGE"`
}

func (c Config) Validate() error {
	if len(c.AllowedOrigins) == 0 {
		return errors.New("CORS_ALLOWED_ORIGINS is required; list at least one origin")
	}

	for _, origin := range c.AllowedOrigins {
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

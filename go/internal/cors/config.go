package cors

type Config struct {
	AllowedOrigins   []string `env:"ALLOWED_ORIGINS"`
	AllowedMethods   []string `env:"ALLOWED_METHODS"`
	AllowedHeaders   []string `env:"ALLOWED_HEADERS"`
	ExposedHeaders   []string `env:"EXPOSED_HEADERS"`
	AllowCredentials bool     `env:"ALLOW_CREDENTIALS"`
	MaxAge           int      `env:"MAX_AGE"`
}

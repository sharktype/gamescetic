package config

type Config struct {
	// Production is a special flag variable always resolved directly from the real env.
	Production bool `env:"-"`

	Auth      Auth      `envPrefix:"AUTH_"`
	Clients   Clients   `envPrefix:"CLIENTS_"`
	CORS      CORS      `envPrefix:"CORS_"`
	API       API       `envPrefix:"API_"`
	Telemetry Telemetry `envPrefix:"TELEMETRY_"`

	Port    int  `env:"PORT" envDefault:"9500"`
	Verbose bool `env:"VERBOSE" envDefault:"false"`
}

type Auth struct {
	JWKSURL  string `env:"JWKSURL,required,notEmpty"`
	Issuer   string `env:"ISSUER,required,notEmpty"`
	Audience string `env:"AUDIENCE,required,notEmpty"`
}

type Clients struct {
	DB DB `envPrefix:"DB_"`
}

type DB struct {
	DSN string `env:"DSN,required,notEmpty"`
}

type CORS struct {
	AllowedOrigins   []string `env:"ALLOWED_ORIGINS,required,notEmpty"`
	AllowedMethods   []string `env:"ALLOWED_METHODS,required,notEmpty"`
	AllowedHeaders   []string `env:"ALLOWED_HEADERS,required,notEmpty"`
	ExposedHeaders   []string `env:"EXPOSE_HEADERS,required,notEmpty"`
	AllowCredentials bool     `env:"ALLOW_CREDENTIALS,required,notEmpty"`
	MaxAge           int      `env:"MAX_AGE,required,notEmpty"`
}

type API struct {
	Title   string `env:"TITLE" envDefault:"satchel-api"`
	Version string `env:"VERSION" envDefault:"0.1.0"`
}

type Telemetry struct {
	SentryDSN    string            `env:"SENTRY_DSN" envDefault:""`
	OTLPEndpoint string            `env:"OTLP_ENDPOINT" envDefault:""`
	OTLPHeaders  map[string]string `env:"OTLP_HEADERS" envKeyValSeparator:"="`
}

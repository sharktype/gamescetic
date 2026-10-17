package config

type Config struct {
	Auth    Auth    `envPrefix:"AUTH_"`
	Clients Clients `envPrefix:"CLIENTS_"`
	CORS    CORS    `envPrefix:"CORS_"`
	API     API     `envPrefix:"API_"`
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
	Driver string `env:"DRIVER,required,notEmpty"`
	DSN    string `env:"DSN,required,notEmpty"`
}

type CORS struct {
	AllowedOrigins   []string `env:"ALLOWED_ORIGINS"`
	AllowedMethods   []string `env:"ALLOWED_METHODS"`
	AllowedHeaders   []string `env:"ALLOWED_HEADERS"`
	ExposedHeaders   []string `env:"EXPOSED_HEADERS"`
	AllowCredentials bool     `env:"ALLOW_CREDENTIALS"`
	MaxAge           int      `env:"MAX_AGE"`
}

type API struct {
	Title   string `env:"TITLE" envDefault:"satchel-api"`
	Version string `env:"VERSION" envDefault:"0.1.0"`
}

package auth

type Config struct {
	JWKSURL  string `env:"JWKSURL"`
	Issuer   string `env:"ISSUER"`
	Audience string `env:"AUDIENCE"`
}

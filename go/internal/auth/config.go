package auth

type Config struct {
	JWKSURL  string `env:"JWKSURL,required,notEmpty"`
	Issuer   string `env:"ISSUER,required,notEmpty"`
	Audience string `env:"AUDIENCE,required,notEmpty"`
}

package db

type Config struct {
	Driver *string `env:"DRIVER"`
	DSN    *string `env:"DSN"`
}

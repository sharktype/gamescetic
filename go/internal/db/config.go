package db

type Config struct {
	Driver string `env:"DRIVER,required,notEmpty"`
	DSN    string `env:"DSN,required,notEmpty"`
}

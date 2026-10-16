package clients

type Config struct {
	DB *DB `envPrefix:"DB_"`
}

type DB struct {
	Driver *string `env:"DRIVER"`
	DSN    *string `env:"DSN"`
}

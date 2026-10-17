package clients

import "github.com/sharktype/satchel/internal/db"

type Config struct {
	DB *db.Config `envPrefix:"DB_" env:",init"`
}

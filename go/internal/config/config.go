package config

import (
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/cors"
)

type Config struct {
	Auth    auth.Config    `envPrefix:"AUTH_"`
	Clients clients.Config `envPrefix:"CLIENTS_"`
	CORS    cors.Config    `envPrefix:"CORS_"`

	API API `envPrefix:"API_"`
}

type API struct {
	Title   string `env:"TITLE" envDefault:"satchel-api"`
	Version string `env:"VERSION" envDefault:"0.1.0"`
}

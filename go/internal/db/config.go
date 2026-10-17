package db

type Config struct {
	Driver string

	// DSN is the Postgres connection string; use sslmode=verify-full in production.
	DSN string
}

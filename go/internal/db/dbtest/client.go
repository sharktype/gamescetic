package dbtest

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/sql"
)

var migrator = goosemigrator.New("migrations", goosemigrator.WithFS(sql.Migrations))

var loadServerConfig = sync.OnceValues(func() (pgtestdb.Config, error) {
	production, err := config.IsProduction()
	if err != nil {
		return pgtestdb.Config{}, err
	}

	if production {
		return pgtestdb.Config{}, errors.New(
			"refusing to create test databases while PRODUCTION is set",
		)
	}

	if err := config.LoadEnv(); err != nil {
		return pgtestdb.Config{}, fmt.Errorf("load .env: %w", err)
	}

	// A test suite passing without database-enabled tests passing proves exactly nothing.

	dsn := os.Getenv("CLIENTS_DB_TEST_DSN")
	if dsn == "" {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_TEST_DSN is not set")
	}

	return serverConfig(dsn)
})

// New returns a client for a fresh, migrated database.
//
// The database is dropped if the test passes and kept for inspection if it fails (logs for trace).
func New(t testing.TB) *db.Store {
	t.Helper()

	return db.New(newPool(t))
}

// Provision returns a client for a new database containing the user from ProvisionedUser.
func Provision(t testing.TB) *db.Store {
	t.Helper()

	pool := newPool(t)
	user := ProvisionedUser()

	_, err := pool.Exec(
		t.Context(),
		`INSERT INTO "user" (id, sub, name, email) VALUES ($1, $2, $3, $4)`,
		user.ID,
		user.Sub,
		user.Name,
		user.Email,
	)
	if err != nil {
		t.Fatal(err)
	}

	return db.New(pool)
}

// ProvisionedUser returns the user Provision inserts. Only ID, Sub, Name, and Email are set.
func ProvisionedUser() generated.User {
	return generated.User{
		ID:    "user-1",
		Sub:   "sub-1",
		Name:  "Ada Lovelace",
		Email: "ada-lovelace@example.com",
	}
}

// NewClosed returns a client whose pool is already closed, so every call fails.
func NewClosed(t testing.TB) *db.Store {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), "postgres://closed")
	if err != nil {
		t.Fatal(err)
	}

	pool.Close()

	return db.New(pool)
}

func newPool(t testing.TB) *pgxpool.Pool {
	t.Helper()

	config, err := loadServerConfig()
	if err != nil {
		t.Fatal(err)
	}

	instance := pgtestdb.Custom(t, config, migrator)

	pool, err := pgxpool.New(t.Context(), instance.URL())
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup runs functions in reverse order of registration which means the pool closes before
	// pgtestdb cleans up (and therefore drops) the database. Postgres will not allow dropping a
	// database that still has test peeps connected to it.

	t.Cleanup(pool.Close)

	return pool
}

// serverConfig parses a DSN into pgtestdb's config.
//
// The DSN's DB has to exist and its user must be able to create roles and DBs.
func serverConfig(dsn string) (pgtestdb.Config, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return pgtestdb.Config{}, fmt.Errorf("parse CLIENTS_DB_TEST_DSN: %w", err)
	}

	password, _ := parsed.User.Password()

	return pgtestdb.Config{
		DriverName: "pgx",
		Host:       parsed.Hostname(),
		Port:       cmp.Or(parsed.Port(), "5432"),
		User:       parsed.User.Username(),
		Password:   password,
		Database:   strings.TrimPrefix(parsed.Path, "/"),
		Options:    parsed.RawQuery,
	}, nil
}

package dbtest

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/db"
)

type Client struct {
	*db.Store

	pool *pgxpool.Pool
}

type server struct {
	config   pgtestdb.Config
	migrator pgtestdb.Migrator
}

var loadServer = sync.OnceValues(func() (server, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return server{}, errors.New("failed to locate dbtest")
	}

	// Yes, this is ugly, but it gets us reliably to root and fails otherwise, with no destructive
	// side effects, so it's reasonable enough to not write helper functions.

	root := filepath.Join(filepath.Dir(file), "..", "..", "..")

	err := godotenv.Load(filepath.Join(root, ".env"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return server{}, fmt.Errorf("load .env: %w", err)
	}

	config, err := serverConfig(os.Getenv("CLIENTS_DB_DSN"))
	if err != nil {
		return server{}, err
	}

	return server{
		config: config,
		migrator: goosemigrator.New(
			"migrations", goosemigrator.WithFS(os.DirFS(filepath.Join(root, "sql"))),
		),
	}, nil
})

// New returns a client for a fresh, migrated database.
//
// The database is dropped if the test passes and kept for inspection if it fails (logs for trace).
func New(t testing.TB) *Client {
	t.Helper()

	srv, err := loadServer()
	if err != nil {
		t.Fatal(err)
	}

	instance := pgtestdb.Custom(t, srv.config, srv.migrator)

	pool, err := pgxpool.New(t.Context(), instance.URL())
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup runs functions in reverse order of registration, which means the pool closes before
	// pgtestdb cleans up (and therefore drops) the database. Postgres will not allow dropping a
	// database that still has test peeps connected to it.

	t.Cleanup(pool.Close)

	return &Client{
		Store: db.New(pool),
		pool:  pool,
	}
}

// serverConfig parses a DSN into pgtestdb's config.
//
// The DSN's DB only has to exist. Its user must be able to create roles and databases. This is
// only true for environments where the tests have to run; production environments should have
// the DB user scoped to a specific database.
//
// Warning: this is not enforced by the service code, but then again, neither are good security
// practices in general. So as always: use your best judgement.
func serverConfig(dsn string) (pgtestdb.Config, error) {
	if dsn == "" {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_DSN is not set")
	}

	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_DSN must be a postgres:// URL")
	}

	port := parsed.Port()
	if port == "" {
		port = "5432"
	}

	password, _ := parsed.User.Password()

	return pgtestdb.Config{
		DriverName: "pgx",
		Host:       parsed.Hostname(),
		Port:       port,
		User:       parsed.User.Username(),
		Password:   password,
		Database:   strings.TrimPrefix(parsed.Path, "/"),
		Options:    parsed.RawQuery,
	}, nil
}

// Unavailable returns a client whose every call fails, standing in for an unreachable database.
func Unavailable(t testing.TB) *db.Store {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), "postgres://closed")
	if err != nil {
		t.Fatal(err)
	}

	pool.Close()

	return db.New(pool)
}

// ProvisionUser inserts the user every test token's "sub-1" resolves to.
func (c *Client) ProvisionUser(t testing.TB) generated.User {
	t.Helper()

	user := generated.User{
		ID:    "user-1",
		Sub:   "sub-1",
		Name:  "Ada Lovelace",
		Email: "ada-lovelace@example.com",
	}

	_, err := c.pool.Exec(
		t.Context(),
		`INSERT INTO "user" (id, sub, name, email) VALUES ($1, $2, $3, $4)`,
		user.ID, user.Sub, user.Name, user.Email,
	)
	if err != nil {
		t.Fatal(err)
	}

	return user
}

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

	config, err := testConfig(
		os.Getenv("CLIENTS_DB_TEST_DSN"),
		os.Getenv("CLIENTS_DB_DSN"),
		os.Getenv("PRODUCTION") != "",
	)
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

// Provision returns a client for a new database containing the user with the "sub-1" subject.
func Provision(t testing.TB) (*Client, generated.User) {
	t.Helper()

	client := New(t)

	return client, client.insertUser(t)
}

// testConfig refuses to guess which database tests may create databases on.
func testConfig(testDSN, appDSN string, production bool) (pgtestdb.Config, error) {
	if production {
		return pgtestdb.Config{}, errors.New(
			"refusing to create test databases while PRODUCTION is set",
		)
	}

	if testDSN == "" {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_TEST_DSN is not set")
	}

	if testDSN == appDSN {
		return pgtestdb.Config{}, errors.New(
			"CLIENTS_DB_TEST_DSN must differ from CLIENTS_DB_DSN",
		)
	}

	return serverConfig(testDSN)
}

// serverConfig parses a DSN into pgtestdb's config.
//
// The DSN's DB only has to exist. Its user must be able to create roles and databases. This is
// only true for environments where the tests have to run; production environments should have
// the DB user scoped to a specific database.
func serverConfig(dsn string) (pgtestdb.Config, error) {
	if dsn == "" {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_TEST_DSN is not set")
	}

	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_TEST_DSN must be a postgres:// URL")
	}

	user := parsed.User.Username()
	password, _ := parsed.User.Password()

	if user == "" {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_TEST_DSN must name a user")
	}

	if parsed.Hostname() == "" {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_TEST_DSN must name a host")
	}

	database := strings.TrimPrefix(parsed.Path, "/")
	if database == "" {
		return pgtestdb.Config{}, errors.New("CLIENTS_DB_TEST_DSN must name a database")
	}

	// pgtestdb reassembles these into its own URL without escaping them, so credentials that need
	// escaping are rejected instead of being silently mangled.

	if strings.ContainsAny(user+password, ":@/?#%") {
		return pgtestdb.Config{}, errors.New(
			"CLIENTS_DB_TEST_DSN credentials must not contain : @ / ? # or %",
		)
	}

	port := parsed.Port()
	if port == "" {
		port = "5432"
	}

	return pgtestdb.Config{
		DriverName: "pgx",
		Host:       parsed.Hostname(),
		Port:       port,
		User:       user,
		Password:   password,
		Database:   database,
		Options:    parsed.RawQuery,
	}, nil
}

// NewUnavailable returns a client whose every call fails, standing in for an unreachable database.
func NewUnavailable(t testing.TB) *db.Store {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), "postgres://closed")
	if err != nil {
		t.Fatal(err)
	}

	pool.Close()

	return db.New(pool)
}

// insertUser inserts the user every test token's "sub-1" resolves to and returns it as stored.
func (c *Client) insertUser(t testing.TB) generated.User {
	t.Helper()

	var user generated.User

	err := c.pool.QueryRow(
		t.Context(),
		`INSERT INTO "user" (id, sub, name, email) VALUES ($1, $2, $3, $4)
		RETURNING id, sub, name, email, avatar_url, created_at, updated_at`,
		"user-1", "sub-1", "Ada Lovelace", "ada-lovelace@example.com",
	).Scan(
		&user.ID, &user.Sub, &user.Name, &user.Email, &user.AvatarUrl, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	return user
}

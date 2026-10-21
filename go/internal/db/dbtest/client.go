package dbtest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/sharktype/satchel/internal/db"
)

type Client struct {
	*db.Store

	pool *pgxpool.Pool
}

func New(t testing.TB) *Client {
	t.Helper()

	loadDotEnv(t)

	dsn := os.Getenv("CLIENTS_DB_TEST_DSN")
	if dsn == "" {
		t.Fatal("CLIENTS_DB_TEST_DSN is not set")
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	err = pool.Ping(t.Context())
	if hasCode(err, "3D000") {
		t.Fatal("the testing database does not exist")
	}

	if err != nil {
		t.Fatal(err)
	}

	migrated := false

	err = pool.QueryRow(
		t.Context(), "SELECT to_regclass('goose_db_version') IS NOT NULL",
	).Scan(&migrated)
	if err != nil {
		t.Fatal(err)
	}

	if !migrated {
		t.Fatal("the testing database has no migrations applied")
	}

	client := &Client{
		Store: db.New(pool),
		pool:  pool,
	}

	client.CleanupAfter(t)

	return client
}

func loadDotEnv(t testing.TB) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate .env")
	}

	err := godotenv.Load(filepath.Join(filepath.Dir(file), "..", "..", "..", ".env"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("load .env: %v", err)
	}
}

func (c *Client) CleanupAfter(tb testing.TB) {
	tb.Helper()
	tb.Cleanup(func() {
		if err := c.Cleanup(context.Background()); err != nil {
			tb.Errorf("cleanup testing database: %v", err)
		}
	})
}

// Cleanup truncates every table in the public schema, leaving goose's bookkeeping alone.
func (c *Client) Cleanup(ctx context.Context) error {
	rows, err := c.pool.Query(ctx, `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public' AND tablename <> 'goose_db_version'
		ORDER BY tablename
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var tables []string

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return err
		}

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if len(tables) == 0 {
		return nil
	}

	identifiers := make([]string, len(tables))
	for i, table := range tables {
		identifiers[i] = pgx.Identifier{table}.Sanitize()
	}

	_, err = c.pool.Exec(
		ctx, "TRUNCATE TABLE "+strings.Join(identifiers, ", ")+" RESTART IDENTITY CASCADE",
	)

	return err
}

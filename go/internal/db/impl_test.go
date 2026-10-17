package db

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sqlc "github.com/sharktype/satchel/db"
)

const testDatabaseName = "testing"

type TestClient struct {
	*store

	// pool is the connection pool for the testing database, used for cleanup.
	pool *pgxpool.Pool
}

func NewTestClient(t testing.TB, dsn string) *TestClient {
	t.Helper()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}

	config.ConnConfig.Database = testDatabaseName

	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.Ping(t.Context()); err != nil {
		t.Fatal(err)
	}

	database := &TestClient{
		store: &store{
			Queries: sqlc.New(pool),
			db:      pool,
		},
		pool: pool,
	}

	t.Cleanup(func() {
		if err := database.Cleanup(context.Background()); err != nil {
			t.Errorf("cleanup testing database: %v", err)
		}

		pool.Close()
	})

	return database
}

// Cleanup truncates every table in the public schema, leaving goose's bookkeeping alone.
func (c *TestClient) Cleanup(ctx context.Context) error {
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

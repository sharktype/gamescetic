package db_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharktype/satchel/internal/db"
)

const databaseName = "testing"

type Client struct {
	db.Client

	pool *pgxpool.Pool
}

func New(t testing.TB, dsn string) *Client {
	t.Helper()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}

	config.ConnConfig.Database = databaseName

	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.Ping(t.Context()); err != nil {
		t.Fatal(err)
	}

	client := &Client{
		Client: db.New(pool),
		pool:   pool,
	}

	t.Cleanup(pool.Close)
	client.CleanupAfter(t)

	return client
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

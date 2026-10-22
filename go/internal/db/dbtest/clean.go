package dbtest

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Clean drops every database pgtestdb has created, including the template.
//
// It must only run while no tests do since that would drop their databases out from under them.
// While funny, that's not really desirable for tests.
func Clean(ctx context.Context) error {
	srv, err := loadServer()
	if err != nil {
		return err
	}

	conn, err := pgx.Connect(ctx, srv.config.URL())
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	rows, err := conn.Query(ctx, `SELECT datname FROM pg_database WHERE datname LIKE 'testdb\_tpl\_%'`)
	if err != nil {
		return err
	}

	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}

	for _, name := range names {
		identifier := pgx.Identifier{name}.Sanitize()

		// Postgres refuses to drop a database while it is marked as a template.

		if _, err := conn.Exec(
			ctx, fmt.Sprintf("ALTER DATABASE %s IS_TEMPLATE false", identifier),
		); err != nil {
			return fmt.Errorf("unmark template %s: %w", name, err)
		}

		if _, err := conn.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", identifier)); err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
	}

	return nil
}

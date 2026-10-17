package db_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	sqlc "github.com/sharktype/satchel/db"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/db"
	dbtest "github.com/sharktype/satchel/internal/db-test"
	"github.com/sharktype/satchel/internal/logger"
	"go.uber.org/zap"
)

func TestNewPool(t *testing.T) {
	// These strings are up here to workaround a Zed issue where syntax highlighting goes wonky.

	const postgresConnectString = "postgres://localhost:5432/test"
	const mysqlConnectString = "mysql://localhost:3306/test"

	// Note that this test does not actually connect to the database.

	tests := []struct {
		name        string
		config      *db.Config
		expectedErr string
	}{
		{
			name:        "no config",
			config:      nil,
			expectedErr: "database connection is required",
		},
		{
			name:        "missing driver",
			config:      &db.Config{DSN: new(postgresConnectString)},
			expectedErr: "database connection is required",
		},
		{
			name:        "missing dsn",
			config:      &db.Config{Driver: new("postgres")},
			expectedErr: "database connection is required",
		},
		{
			name:        "unsupported driver",
			config:      &db.Config{Driver: new("mysql"), DSN: new(mysqlConnectString)},
			expectedErr: "unsupported database driver: mysql",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := db.NewPool(context.Background(), test.config, zap.NewNop())

			if client != nil {
				t.Fatal("expected no client")
			}

			if err == nil || !strings.Contains(err.Error(), test.expectedErr) {
				t.Fatalf("got error %v, want it to contain %q", err, test.expectedErr)
			}
		})
	}
}

func TestExecTx(t *testing.T) {
	t.Setenv("PRODUCTION", "1")

	cfg := config.MustNew(logger.MustNew())
	if cfg.Clients.DB == nil || cfg.Clients.DB.DSN == nil {
		t.Skip("CLIENTS_DB_DSN is not set")
	}

	database := dbtest.New(t, *cfg.Clients.DB.DSN)

	t.Run("commits on success", func(t *testing.T) {
		database.CleanupAfter(t)

		err := database.ExecTx(t.Context(), func(tx db.Client) error {
			return tx.SetMeta(t.Context(), sqlc.SetMetaParams{
				Key:   "meta-1",
				Value: "value-1",
			})
		})
		if err != nil {
			t.Fatal(err)
		}

		value, err := database.GetMeta(t.Context(), "meta-1")
		if err != nil {
			t.Fatal(err)
		}

		if value != "value-1" {
			t.Fatalf("got value %q, want %q", value, "value-1")
		}
	})

	t.Run("rolls back when the callback fails", func(t *testing.T) {
		database.CleanupAfter(t)

		boom := errors.New("boom")

		err := database.ExecTx(t.Context(), func(tx db.Client) error {
			if err := tx.SetMeta(t.Context(), sqlc.SetMetaParams{
				Key:   "meta-1",
				Value: "value-1",
			}); err != nil {
				return err
			}

			return boom
		})
		if !errors.Is(err, boom) {
			t.Fatalf("got error %v, want %v", err, boom)
		}

		if _, err := database.GetMeta(t.Context(), "meta-1"); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("got error %v, want %v", err, pgx.ErrNoRows)
		}
	})

	t.Run("does not nest", func(t *testing.T) {
		database.CleanupAfter(t)

		err := database.ExecTx(t.Context(), func(outer db.Client) error {
			if !outer.IsTx() {
				t.Error("expected the outer callback to be transactional")
			}

			return outer.ExecTx(t.Context(), func(inner db.Client) error {
				if !inner.IsTx() {
					t.Error("expected the inner callback to be transactional")
				}

				return inner.SetMeta(t.Context(), sqlc.SetMetaParams{
					Key:   "meta-1",
					Value: "value-1",
				})
			})
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := database.GetMeta(t.Context(), "meta-1"); err != nil {
			t.Fatalf("expected the nested write to be committed: %v", err)
		}
	})
}

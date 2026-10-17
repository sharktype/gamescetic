package db_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/config"
	"github.com/sharktype/satchel/internal/db"
	test_db "github.com/sharktype/satchel/internal/db/test-db"
)

func TestExecTx(t *testing.T) {
	// Intentional: we must use the same DSN config for testing as the application, and we also
	// must parse the entire config for tests to keep config consistent.

	cfg, err := config.New()
	if err != nil {
		t.Fatal(err)
	}

	database := test_db.New(t, cfg.Clients.DB.DSN)

	t.Run("commits on success", func(t *testing.T) {
		database.CleanupAfter(t)

		err := database.ExecTx(t.Context(), func(tx db.Client) error {
			return tx.SetMeta(t.Context(), generated.SetMetaParams{
				Key:   "key-1",
				Value: "value-1",
			})
		})
		if err != nil {
			t.Fatal(err)
		}

		value, err := database.GetMeta(t.Context(), "key-1")
		if err != nil {
			t.Fatal(err)
		}

		if value != "value-1" {
			t.Fatalf("got value %q, want %q", value, "value-1")
		}
	})

	t.Run("rolls back when the callback fails", func(t *testing.T) {
		database.CleanupAfter(t)

		expectedErr := errors.New("arbitrary error")

		err := database.ExecTx(t.Context(), func(tx db.Client) error {
			if err := tx.SetMeta(t.Context(), generated.SetMetaParams{
				Key:   "key-1",
				Value: "value-1",
			}); err != nil {
				return err
			}

			return expectedErr
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("got error %v, want %v", err, expectedErr)
		}

		if _, err := database.GetMeta(t.Context(), "key-1"); !errors.Is(err, pgx.ErrNoRows) {
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

				return inner.SetMeta(t.Context(), generated.SetMetaParams{
					Key:   "key-1",
					Value: "value-1",
				})
			})
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, err := database.GetMeta(t.Context(), "key-1"); err != nil {
			t.Fatalf("expected the nested write to be committed: %v", err)
		}
	})
}

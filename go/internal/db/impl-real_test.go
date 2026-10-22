package db_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/db/dbtest"
)

func TestExecTx(t *testing.T) {
	t.Run("commits on success", func(t *testing.T) {
		database := dbtest.New(t)

		err := database.ExecTx(t.Context(), func(tx generated.Querier) error {
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
		database := dbtest.New(t)

		expectedErr := errors.New("arbitrary error")

		err := database.ExecTx(t.Context(), func(tx generated.Querier) error {
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
}

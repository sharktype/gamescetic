package db

import (
	"context"
	"testing"
)

func TestNewPool(t *testing.T) {
	t.Run("an unparseable DSN is rejected", func(t *testing.T) {
		pool, err := NewPool(context.Background(), Config{DSN: "not-a-dsn"})

		if pool != nil {
			t.Fatal("expected no pool")
		}

		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

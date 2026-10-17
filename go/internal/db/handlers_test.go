package db

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNewPool(t *testing.T) {
	pool, err := newPool(
		context.Background(),
		zap.NewNop(),
		"mysql",
		"mysql://localhost:3306/test",
	)

	if pool != nil {
		t.Fatal("expected no pool")
	}

	if err == nil || !strings.Contains(err.Error(), "unsupported database driver: mysql") {
		t.Fatalf("got error %v, want it to contain %q", err, "unsupported database driver: mysql")
	}
}

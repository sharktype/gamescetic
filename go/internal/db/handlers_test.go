package db

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNewPool(t *testing.T) {
	// These strings are up here to workaround a Zed issue where syntax highlighting goes wonky.

	const postgresConnectString = "postgres://localhost:5432/test"
	const mysqlConnectString = "mysql://localhost:3306/test"

	// Note that this test does not actually connect to the database.

	tests := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name:        "no config",
			config:      nil,
			expectedErr: "database connection is required",
		},
		{
			name:        "missing driver",
			config:      &Config{DSN: new(postgresConnectString)},
			expectedErr: "database connection is required",
		},
		{
			name:        "missing dsn",
			config:      &Config{Driver: new("postgres")},
			expectedErr: "database connection is required",
		},
		{
			name:        "unsupported driver",
			config:      &Config{Driver: new("mysql"), DSN: new(mysqlConnectString)},
			expectedErr: "unsupported database driver: mysql",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool, err := newPool(context.Background(), test.config, zap.NewNop())

			if pool != nil {
				t.Fatal("expected no pool")
			}

			if err == nil || !strings.Contains(err.Error(), test.expectedErr) {
				t.Fatalf("got error %v, want it to contain %q", err, test.expectedErr)
			}
		})
	}
}

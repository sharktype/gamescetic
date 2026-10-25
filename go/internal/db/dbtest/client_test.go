package dbtest

import (
	"testing"

	"github.com/peterldowns/pgtestdb"
)

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expected    pgtestdb.Config
		expectedErr bool
	}{
		{
			name: "every part is read",
			dsn:  "postgres://ada:secret@db.example.com:6543/gamescetic?sslmode=verify-full",
			expected: pgtestdb.Config{
				DriverName: "pgx",
				Host:       "db.example.com",
				Port:       "6543",
				User:       "ada",
				Password:   "secret",
				Database:   "gamescetic",
				Options:    "sslmode=verify-full",
			},
		},
		{
			name: "default port",
			dsn:  "postgresql://ada:secret@localhost/gamescetic",
			expected: pgtestdb.Config{
				DriverName: "pgx",
				Host:       "localhost",
				Port:       "5432",
				User:       "ada",
				Password:   "secret",
				Database:   "gamescetic",
			},
		},
		{
			name: "password is optional",
			dsn:  "postgres://ada@localhost/gamescetic",
			expected: pgtestdb.Config{
				DriverName: "pgx",
				Host:       "localhost",
				Port:       "5432",
				User:       "ada",
				Database:   "gamescetic",
			},
		},
		{name: "unparseable URL", dsn: "postgres://%zz", expectedErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := serverConfig(test.dsn)

			if test.expectedErr != (err != nil) {
				t.Fatalf("got error %v, want error %t", err, test.expectedErr)
			}

			if got != test.expected {
				t.Fatalf("got %+v, want %+v", got, test.expected)
			}
		})
	}
}

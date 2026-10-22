package dbtest

import (
	"testing"

	"github.com/peterldowns/pgtestdb"
)

func TestTestConfig(t *testing.T) {
	const (
		testDSN = "postgres://ada:secret@localhost/satchel_test"
		appDSN  = "postgres://satchel:satchel@localhost/satchel"
	)

	expected := pgtestdb.Config{
		DriverName: "pgx",
		Host:       "localhost",
		Port:       "5432",
		User:       "ada",
		Password:   "secret",
		Database:   "satchel_test",
	}

	tests := []struct {
		name        string
		testDSN     string
		appDSN      string
		production  bool
		expected    pgtestdb.Config
		expectedErr bool
	}{
		{name: "uses the test DSN", testDSN: testDSN, appDSN: appDSN, expected: expected},
		{
			name:        "refuses in production",
			testDSN:     testDSN,
			appDSN:      appDSN,
			production:  true,
			expectedErr: true,
		},
		{name: "refuses the service DSN", testDSN: testDSN, appDSN: testDSN, expectedErr: true},
		{name: "requires a test DSN", appDSN: appDSN, expectedErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := testConfig(test.testDSN, test.appDSN, test.production)

			if test.expectedErr != (err != nil) {
				t.Fatalf("got error %v, want error %t", err, test.expectedErr)
			}

			if got != test.expected {
				t.Fatalf("got %+v, want %+v", got, test.expected)
			}
		})
	}
}

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expected    pgtestdb.Config
		expectedErr bool
	}{
		{
			name: "every part is read",
			dsn:  "postgres://ada:secret@db.example.com:6543/satchel?sslmode=verify-full",
			expected: pgtestdb.Config{
				DriverName: "pgx",
				Host:       "db.example.com",
				Port:       "6543",
				User:       "ada",
				Password:   "secret",
				Database:   "satchel",
				Options:    "sslmode=verify-full",
			},
		},
		{
			name: "postgresql scheme and default port",
			dsn:  "postgresql://ada:secret@localhost/satchel",
			expected: pgtestdb.Config{
				DriverName: "pgx",
				Host:       "localhost",
				Port:       "5432",
				User:       "ada",
				Password:   "secret",
				Database:   "satchel",
			},
		},
		{
			name: "password is optional",
			dsn:  "postgres://ada@localhost/satchel",
			expected: pgtestdb.Config{
				DriverName: "pgx",
				Host:       "localhost",
				Port:       "5432",
				User:       "ada",
				Database:   "satchel",
			},
		},
		{name: "empty DSN", expectedErr: true},
		{
			name:        "not a postgres URL",
			dsn:         "mysql://ada:secret@localhost/satchel",
			expectedErr: true,
		},
		{name: "unparseable URL", dsn: "postgres://%zz", expectedErr: true},
		{name: "no user", dsn: "postgres://localhost/satchel", expectedErr: true},
		{name: "no host", dsn: "postgres://ada@/satchel", expectedErr: true},
		{
			name:        "IPv6 host",
			dsn:         "postgres://ada:secret@[::1]:5432/satchel",
			expectedErr: true,
		},
		{name: "no database", dsn: "postgres://ada@localhost", expectedErr: true},
		{name: "escaped @ in password", dsn: "postgres://ada:p%40ss@localhost/satchel", expectedErr: true},
		{
			name:        "escaped : in user",
			dsn:         "postgres://ad%3Aa:secret@localhost/satchel",
			expectedErr: true,
		},
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

package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/internal/db"
)

func TestRequireUser(t *testing.T) {
	arbitraryErr := errors.New("arbitrary error")

	tests := []struct {
		name        string
		row         []any
		queryErr    error
		expectedID  string
		expectedErr error
	}{
		{
			name:       "provisioned user",
			row:        []any{"user-1", "sub-1", "Ada", "ada@lovelace.com", nil, time.Now(), nil},
			expectedID: "user-1",
		},
		{
			name:        "unprovisioned user",
			queryErr:    pgx.ErrNoRows,
			expectedErr: ErrUserNotProvisioned,
		},
		{
			name:        "lookup failure",
			queryErr:    arbitraryErr,
			expectedErr: arbitraryErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := db.NewStub()
			stub.Row = test.row
			stub.QueryRowErr = test.queryErr

			user, err := NewManager(stub).RequireUser(context.Background(), "sub-1")

			if test.expectedErr != nil {
				if !errors.Is(err, test.expectedErr) {
					t.Fatalf("got error %v, want %v", err, test.expectedErr)
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if user.ID != test.expectedID {
				t.Fatalf("got user %q, want %q", user.ID, test.expectedID)
			}

			if user.Email != "ada@example.com" {
				t.Fatalf("got email %q, want %q", user.Email, "ada@example.com")
			}
		})
	}
}

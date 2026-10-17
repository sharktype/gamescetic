package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sharktype/satchel/generated"
	test_db "github.com/sharktype/satchel/internal/db/test-db"
)

func TestRequireUser(t *testing.T) {
	arbitraryErr := errors.New("arbitrary error")

	provisioned := generated.User{
		ID:    "user-1",
		Sub:   "sub-1",
		Name:  "Ada Lovelace",
		Email: "ada-lovelace@example.com",
	}

	tests := []struct {
		name        string
		user        generated.User
		userErr     error
		expectedErr error
	}{
		{
			name: "provisioned user",
			user: provisioned,
		},
		{
			name:        "unprovisioned user",
			userErr:     pgx.ErrNoRows,
			expectedErr: ErrUserNotProvisioned,
		},
		{
			name:        "lookup failure",
			userErr:     arbitraryErr,
			expectedErr: arbitraryErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := &test_db.Stub{User: test.user, UserErr: test.userErr}

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

			if user != test.user {
				t.Fatalf("got user %+v, want %+v", user, test.user)
			}
		})
	}
}

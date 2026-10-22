package auth_test

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/auth/authtest"
	"github.com/sharktype/satchel/internal/db"
	"github.com/sharktype/satchel/internal/db/dbtest"
)

func TestNewJWTAuthMiddleware(t *testing.T) {
	t.Run("a verified token resolves the user by its subject", func(t *testing.T) {
		row := dbtest.ProvisionedUser

		api, signer := newTestAPI(t, dbtest.Provision(t))
		probe := authtest.RegisterProbe(api, authtest.RequireJWT())

		token := signer.Token(t, row.Sub)

		code := api.Get(authtest.ProbePath, "Authorization: Bearer "+token).Code
		if code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		expected := auth.User{
			ID:    row.ID,
			Sub:   row.Sub,
			Name:  row.Name,
			Email: row.Email,
		}

		if probe.User != expected {
			t.Fatalf("got user %+v, want %+v", probe.User, expected)
		}
	})

	rejections := []struct {
		name           string
		database       func(testing.TB) *db.Store
		token          func(t *testing.T, signer *authtest.Signer) string
		expectedStatus int
	}{
		{
			name:     "a token signed by another key is rejected",
			database: dbtest.NewClosed,
			token: func(t *testing.T, _ *authtest.Signer) string {
				return authtest.NewSigner(t).Token(t, dbtest.ProvisionedUser.Sub)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "a token without a subject is rejected",
			database: dbtest.NewClosed,
			token: func(t *testing.T, signer *authtest.Signer) string {
				return signer.Token(t, "")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "a subject that is not the provisioned user is forbidden",
			database: dbtest.Provision,
			token: func(t *testing.T, signer *authtest.Signer) string {
				return signer.Token(t, "sub-2")
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:     "a failing user lookup is an internal error",
			database: dbtest.NewClosed,
			token: func(t *testing.T, signer *authtest.Signer) string {
				return signer.Token(t, dbtest.ProvisionedUser.Sub)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range rejections {
		t.Run(test.name, func(t *testing.T) {
			api, signer := newTestAPI(t, test.database(t))
			probe := authtest.RegisterProbe(api, authtest.RequireJWT())

			code := api.Get(
				authtest.ProbePath, "Authorization: Bearer "+test.token(t, signer),
			).Code
			if code != test.expectedStatus {
				t.Fatalf("got %d, want %d", code, test.expectedStatus)
			}

			if probe.Ran {
				t.Fatal("the handler ran")
			}
		})
	}

	t.Run("scopes", func(t *testing.T) {
		// "posts" is just an example here.

		tests := []struct {
			name           string
			declared       []string
			granted        []string
			expectedStatus int
			expectedIsRun  bool
		}{
			{
				name:           "no scopes declared",
				expectedStatus: http.StatusOK,
				expectedIsRun:  true,
			},
			{
				name:           "all declared scopes granted",
				declared:       []string{"posts:read", "posts:write"},
				granted:        []string{"posts:read", "posts:write"},
				expectedStatus: http.StatusOK,
				expectedIsRun:  true,
			},
			{
				name:           "extra scopes are fine",
				declared:       []string{"posts:read"},
				granted:        []string{"posts:read", "admin"},
				expectedStatus: http.StatusOK,
				expectedIsRun:  true,
			},
			{
				name:           "one of two declared scopes is not enough",
				declared:       []string{"posts:read", "posts:write"},
				granted:        []string{"posts:read"},
				expectedStatus: http.StatusForbidden,
			},
			{
				name:           "no scopes claim",
				declared:       []string{"posts:read"},
				expectedStatus: http.StatusForbidden,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				api, signer := newTestAPI(t, dbtest.Provision(t))
				probe := authtest.RegisterProbe(api, authtest.RequireJWT(test.declared...))

				token := signer.TokenWithScopes(t, dbtest.ProvisionedUser.Sub, test.granted)

				code := api.Get(authtest.ProbePath, "Authorization: Bearer "+token).Code
				if code != test.expectedStatus {
					t.Fatalf("got %d, want %d", code, test.expectedStatus)
				}

				if probe.Ran != test.expectedIsRun {
					t.Fatalf("handler ran: got %t, want %t", probe.Ran, test.expectedIsRun)
				}
			})
		}
	})
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name           string
		authorization  func(token string) string
		expectedStatus int
	}{
		{
			name:           "bearer scheme",
			authorization:  func(token string) string { return "Bearer " + token },
			expectedStatus: http.StatusOK,
		},
		{
			name:           "scheme is case insensitive",
			authorization:  func(token string) string { return "bearer " + token },
			expectedStatus: http.StatusOK,
		},
		{
			name:           "surrounding whitespace is trimmed",
			authorization:  func(token string) string { return "Bearer   " + token + " " },
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing scheme",
			authorization:  func(token string) string { return token },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "other scheme",
			authorization:  func(token string) string { return "Basic " + token },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty header",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "scheme without a token",
			authorization:  func(string) string { return "Bearer " },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "embedded whitespace",
			authorization:  func(string) string { return "Bearer foo bar" },
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api, signer := newTestAPI(t, dbtest.Provision(t))
			authtest.RegisterProbe(api, authtest.RequireJWT())

			if test.authorization == nil {
				if code := api.Get(authtest.ProbePath).Code; code != test.expectedStatus {
					t.Fatalf("got %d, want %d", code, test.expectedStatus)
				}

				return
			}

			authorization := test.authorization(signer.Token(t, dbtest.ProvisionedUser.Sub))

			code := api.Get(authtest.ProbePath, "Authorization: "+authorization).Code
			if code != test.expectedStatus {
				t.Fatalf("got %d, want %d", code, test.expectedStatus)
			}
		})
	}
}

func newTestAPI(t *testing.T, users db.Client) (humatest.TestAPI, *authtest.Signer) {
	t.Helper()

	authClient := authtest.New(t, users)

	_, api := humatest.New(t)

	api.UseMiddleware(authClient.Middleware(api))

	return api, authClient.Signer
}

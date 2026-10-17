package auth_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/sharktype/satchel/generated"
	"github.com/sharktype/satchel/internal/auth"
	test_auth "github.com/sharktype/satchel/internal/auth/test-auth"
	"github.com/sharktype/satchel/internal/db"
	test_db "github.com/sharktype/satchel/internal/db/test-db"
	"go.uber.org/zap"
)

var provisioned = generated.User{
	ID:    "user-1",
	Sub:   "sub-1",
	Name:  "Ada Lovelace",
	Email: "ada-lovelace@example.com",
}

func newTestAPI(t *testing.T, idp *test_auth.IDP, users db.Client) humatest.TestAPI {
	t.Helper()

	_, api := humatest.New(t)
	api.UseMiddleware(auth.NewJWTAuthMiddleware(api, idp.Cache, idp.Config(), users, zap.NewNop()))

	return api
}

func TestNewJWTAuthMiddleware(t *testing.T) {
	idp := test_auth.NewIDP(t)

	t.Run("an unsecured operation never resolves a viewer", func(t *testing.T) {
		api := newTestAPI(t, idp, &test_db.Stub{User: provisioned})
		probe := test_auth.RegisterProbe(api, nil)

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
		claims := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"sub-1"}`))

		forged := header + "." + claims + ".no-signature"

		code := api.Get(test_auth.ProbePath, "Authorization: Bearer "+forged).Code
		if code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if probe.Viewer != (generated.User{}) {
			t.Fatalf("got viewer %+v, want none", probe.Viewer)
		}
	})

	t.Run("a verified token resolves the viewer by its subject", func(t *testing.T) {
		api := newTestAPI(t, idp, &test_db.Stub{User: provisioned})
		probe := test_auth.RegisterProbe(api, test_auth.RequireJWT())

		token := idp.Token(t, "sub-1")

		code := api.Get(test_auth.ProbePath, "Authorization: Bearer "+token).Code
		if code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if probe.Viewer != provisioned {
			t.Fatalf("got viewer %+v, want %+v", probe.Viewer, provisioned)
		}
	})

	rejections := []struct {
		name           string
		users          db.Client
		token          string
		expectedStatus int
	}{
		{
			name:           "a token signed by another key is rejected",
			users:          &test_db.Stub{User: provisioned},
			token:          test_auth.NewIDP(t).Token(t, "sub-1"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "a token without a subject is rejected",
			users:          &test_db.Stub{User: provisioned},
			token:          idp.Token(t, ""),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "a subject that is not provisioned is forbidden",
			users:          &test_db.Stub{User: provisioned},
			token:          idp.Token(t, "sub-2"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "a failing user lookup is an internal error",
			users:          &test_db.Stub{UserErr: errors.New("arbitrary error")},
			token:          idp.Token(t, "sub-1"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range rejections {
		t.Run(test.name, func(t *testing.T) {
			api := newTestAPI(t, idp, test.users)
			probe := test_auth.RegisterProbe(api, test_auth.RequireJWT())

			code := api.Get(test_auth.ProbePath, "Authorization: Bearer "+test.token).Code
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
				api := newTestAPI(t, idp, &test_db.Stub{User: provisioned})
				probe := test_auth.RegisterProbe(api, test_auth.RequireJWT(test.declared...))

				token := idp.TokenWithScopes(t, "sub-1", test.granted)

				code := api.Get(test_auth.ProbePath, "Authorization: Bearer "+token).Code
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

func TestMustSingleJWTRequirement(t *testing.T) {
	jwtRequirement := map[string][]string{auth.DefaultJWTSchemeKey: {}}

	tests := []struct {
		name        string
		security    []map[string][]string
		expectPanic bool
	}{
		{
			name: "no security",
		},
		{
			name:     "one JWT requirement",
			security: []map[string][]string{jwtRequirement},
		},
		{
			name:     "one JWT requirement plus another scheme",
			security: []map[string][]string{jwtRequirement, {"APIKey": {}}},
		},
		{
			name:        "two JWT requirements",
			security:    []map[string][]string{jwtRequirement, jwtRequirement},
			expectPanic: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); (recovered != nil) != test.expectPanic {
					t.Fatalf("got panic %v, want panic %t", recovered, test.expectPanic)
				}
			}()

			auth.MustSingleJWTRequirement(nil, &huma.Operation{
				OperationID: "Test",
				Security:    test.security,
			})
		})
	}
}

func TestBearerToken(t *testing.T) {
	idp := test_auth.NewIDP(t)
	token := idp.Token(t, "sub-1")

	tests := []struct {
		name           string
		authorization  string
		expectedStatus int
	}{
		{
			name:           "bearer scheme",
			authorization:  "Bearer " + token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "scheme is case insensitive",
			authorization:  "bearer " + token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "surrounding whitespace is trimmed",
			authorization:  "Bearer   " + token + " ",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing scheme",
			authorization:  token,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "other scheme",
			authorization:  "Basic " + token,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty header",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "scheme without a token",
			authorization:  "Bearer ",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "embedded whitespace",
			authorization:  "Bearer foo bar",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := newTestAPI(t, idp, &test_db.Stub{User: provisioned})
			test_auth.RegisterProbe(api, test_auth.RequireJWT())

			var headers []any
			if test.authorization != "" {
				headers = append(headers, "Authorization: "+test.authorization)
			}

			if code := api.Get(test_auth.ProbePath, headers...).Code; code != test.expectedStatus {
				t.Fatalf("got %d, want %d", code, test.expectedStatus)
			}
		})
	}
}

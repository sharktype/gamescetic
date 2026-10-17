package routing

import (
	"encoding/base64"
	"net/http"
	"testing"
)

func TestNewJWTAuthMiddleware(t *testing.T) {
	idp := newTestIDP(t)

	t.Run("an unsecured operation never sets a subject", func(t *testing.T) {
		router, api := newTestAPI(t, idp, provisionedClients())
		handler := registerProbe(api, nil)

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
		claims := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"attacker"}`))

		code := get(router, probePath, "Bearer "+header+"."+claims+".no-signature").Code
		if code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if handler.subject != "" {
			t.Fatalf("got subject %q, want none", handler.subject)
		}
	})

	t.Run("a verified token sets the subject", func(t *testing.T) {
		router, api := newTestAPI(t, idp, provisionedClients())
		handler := registerProbe(api, requireJWT())

		if code := get(
			router, probePath, "Bearer "+idp.token(t, "user-1"),
		).Code; code != http.StatusOK {
			t.Fatalf("got %d, want %d", code, http.StatusOK)
		}

		if handler.subject != "user-1" {
			t.Fatalf("got subject %q, want %q", handler.subject, "user-1")
		}
	})

	rejections := []struct {
		name  string
		token string
	}{
		{
			name:  "a token signed by another key is rejected",
			token: newTestIDP(t).token(t, "attacker"),
		},
		{
			name:  "a token without a subject is rejected",
			token: idp.token(t, ""),
		},
	}

	for _, test := range rejections {
		t.Run(test.name, func(t *testing.T) {
			router, api := newTestAPI(t, idp, provisionedClients())
			handler := registerProbe(api, requireJWT())

			if code := get(
				router, probePath, "Bearer "+test.token,
			).Code; code != http.StatusUnauthorized {
				t.Fatalf("got %d, want %d", code, http.StatusUnauthorized)
			}

			if handler.ran {
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
				router, api := newTestAPI(t, idp, provisionedClients())
				handler := registerProbe(api, requireJWT(test.declared...))

				token := idp.tokenWithScopes(t, "sub-1", test.granted)

				if code := get(
					router, probePath, "Bearer "+token,
				).Code; code != test.expectedStatus {
					t.Fatalf("got %d, want %d", code, test.expectedStatus)
				}

				if handler.ran != test.expectedIsRun {
					t.Fatalf("handler ran: got %t, want %t", handler.ran, test.expectedIsRun)
				}
			})
		}
	})
}

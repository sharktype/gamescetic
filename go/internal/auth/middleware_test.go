package auth

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestMustSingleJWTRequirement(t *testing.T) {
	jwtRequirement := map[string][]string{DefaultJWTSchemeKey: {}}

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

			MustSingleJWTRequirement(nil, &huma.Operation{
				OperationID: "Test",
				Security:    test.security,
			})
		})
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		header        string
		expectedToken string
		expectedIsOK  bool
	}{
		{
			name:          "bearer scheme",
			header:        "Bearer foobar",
			expectedToken: "foobar",
			expectedIsOK:  true,
		},
		{
			name:          "scheme is case insensitive",
			header:        "bearer foobar",
			expectedToken: "foobar",
			expectedIsOK:  true,
		},
		{
			name:          "surrounding whitespace is trimmed",
			header:        "Bearer   foobar ",
			expectedToken: "foobar",
			expectedIsOK:  true,
		},
		{
			name:          "missing scheme",
			header:        "foobar",
			expectedToken: "",
			expectedIsOK:  false,
		},
		{
			name:          "other scheme",
			header:        "Basic foobar",
			expectedToken: "",
			expectedIsOK:  false,
		},
		{
			name:          "empty header",
			header:        "",
			expectedToken: "",
			expectedIsOK:  false,
		},
		{
			name:          "scheme without a token",
			header:        "Bearer ",
			expectedToken: "",
			expectedIsOK:  false,
		},
		{
			name:          "embedded whitespace",
			header:        "Bearer foo bar",
			expectedToken: "foo bar",
			expectedIsOK:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, ok := bearerToken(test.header)

			if token != test.expectedToken || ok != test.expectedIsOK {
				t.Fatalf("got (%q, %t), want (%q, %t)", token, ok, test.expectedToken, test.expectedIsOK)
			}
		})
	}
}

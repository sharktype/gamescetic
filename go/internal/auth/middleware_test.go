package auth

import (
	"testing"
)

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

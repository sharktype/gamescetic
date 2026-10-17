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
			header:        "Bearer abc.def.ghi",
			expectedToken: "abc.def.ghi",
			expectedIsOK:  true,
		},
		{
			name:          "scheme is case insensitive",
			header:        "bearer abc.def.ghi",
			expectedToken: "abc.def.ghi",
			expectedIsOK:  true,
		},
		{
			name:          "surrounding whitespace is trimmed",
			header:        "Bearer   abc.def.ghi ",
			expectedToken: "abc.def.ghi",
			expectedIsOK:  true,
		},
		{
			name:   "missing scheme",
			header: "abc.def.ghi",
		},
		{
			name:   "other scheme",
			header: "Basic abc",
		},
		{
			name:   "empty header",
			header: "",
		},
		{
			name:   "scheme without a token",
			header: "Bearer ",
		},
		{
			name:   "embedded whitespace",
			header: "Bearer abc def",
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

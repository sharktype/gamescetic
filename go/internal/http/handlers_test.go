package http

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		origins     []string
		expectedErr bool
	}{
		{
			name:    "explicit origins",
			origins: []string{"http://example.com"},
		},
		{
			name:        "no origins specified",
			expectedErr: true,
		},
		{
			name:        "wildcard origin",
			origins:     []string{"*"},
			expectedErr: true,
		},
		{
			name:        "wildcard alongside explicit origins",
			origins:     []string{"*", "http://example.com"},
			expectedErr: true,
		},
		{
			name:        "blank origin",
			origins:     []string{" "},
			expectedErr: true,
		},
		{
			name:    "subdomain wildcard",
			origins: []string{"https://*.example.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validate(test.origins)

			if test.expectedErr != (err != nil) {
				t.Fatalf("got error %v, want error %t", err, test.expectedErr)
			}
		})
	}
}

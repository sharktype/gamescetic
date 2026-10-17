package cors

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectedErr bool
	}{
		{
			name:   "explicit origins",
			config: Config{AllowedOrigins: []string{"http://example.com"}},
		},
		{
			name:        "no origins specified",
			config:      Config{},
			expectedErr: true,
		},
		{
			name:        "wildcard origin",
			config:      Config{AllowedOrigins: []string{"*"}},
			expectedErr: true,
		},
		{
			name:        "wildcard alongside explicit origins",
			config:      Config{AllowedOrigins: []string{"*", "http://example.com"}},
			expectedErr: true,
		},
		{
			name:        "blank origin",
			config:      Config{AllowedOrigins: []string{" "}},
			expectedErr: true,
		},
		{
			name:   "subdomain wildcard",
			config: Config{AllowedOrigins: []string{"https://*.example.com"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()

			if test.expectedErr != (err != nil) {
				t.Fatalf("got error %v, want error %t", err, test.expectedErr)
			}
		})
	}
}

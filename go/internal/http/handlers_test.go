package http

import (
	nethttp "net/http"
	"net/http/httptest"
	"testing"
)

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

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders()(nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, _ *nethttp.Request) {
		writer.WriteHeader(nethttp.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(nethttp.MethodGet, "/", nil))

	expected := map[string]string{
		"Content-Security-Policy": "default-src 'none'",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	}

	for header, value := range expected {
		if got := response.Header().Get(header); got != value {
			t.Fatalf("%s: got %q, want %q", header, got, value)
		}
	}
}

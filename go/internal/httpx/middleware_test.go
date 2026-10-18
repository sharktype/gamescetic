package httpx

import (
	nethttp "net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name        string
		origins     []string
		expected    []string
		expectedErr bool
	}{
		{
			name:     "explicit origins",
			origins:  []string{"http://example.com"},
			expected: []string{"http://example.com"},
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
			name:     "subdomain wildcard",
			origins:  []string{"https://*.example.com"},
			expected: []string{"https://*.example.com"},
		},
		{
			name:     "surrounding whitespace is trimmed",
			origins:  []string{" http://example.com ", "\thttps://example.com"},
			expected: []string{"http://example.com", "https://example.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			origins, err := normalize(test.origins)

			if test.expectedErr != (err != nil) {
				t.Fatalf("got error %v, want error %t", err, test.expectedErr)
			}

			if !slices.Equal(origins, test.expected) {
				t.Fatalf("got origins %v, want %v", origins, test.expected)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders()(
		nethttp.HandlerFunc(func(writer nethttp.ResponseWriter, _ *nethttp.Request) {
			writer.WriteHeader(nethttp.StatusNoContent)
		}),
	)

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

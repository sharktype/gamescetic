package routing

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/auth"
	"github.com/sharktype/satchel/internal/auth/authtest"
)

func TestDeclareJWT(t *testing.T) {
	t.Run("adds a JWT requirement when none is declared", func(t *testing.T) {
		operation := &huma.Operation{OperationID: "Test"}

		declareJWT(operation)

		if len(operation.Security) != 1 {
			t.Fatalf("got %d security requirements, want 1", len(operation.Security))
		}

		if _, ok := operation.Security[0][auth.JWTSchemeKey]; !ok {
			t.Fatal("declareJWT did not add the JWT requirement")
		}
	})

	rejections := []struct {
		name     string
		security []map[string][]string
	}{
		{
			name:     "a non-JWT requirement",
			security: []map[string][]string{{"ApiKey": {}}},
		},
		{
			name:     "an empty requirement",
			security: []map[string][]string{{}},
		},
		{
			name:     "a requirement combining JWT with another scheme",
			security: []map[string][]string{{auth.JWTSchemeKey: {}, "ApiKey": {}}},
		},
		{
			name:     "two JWT requirements",
			security: append(authtest.RequireJWT(), authtest.RequireJWT()...),
		},
	}

	for _, test := range rejections {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected a panic")
				}
			}()

			declareJWT(&huma.Operation{OperationID: "Test", Security: test.security})
		})
	}
}

package utils

import (
	"os"
	"testing"
)

func TestIsProduction(t *testing.T) {
	t.Run("unset is not production", func(t *testing.T) {
		previous, wasSet := os.LookupEnv("PRODUCTION")

		t.Cleanup(func() {
			if wasSet {
				_ = os.Setenv("PRODUCTION", previous)
			}
		})

		_ = os.Unsetenv("PRODUCTION")

		if IsProduction() {
			t.Fatal("got production")
		}
	})

	t.Run(`"1" is production`, func(t *testing.T) {
		t.Setenv("PRODUCTION", "1")

		if !IsProduction() {
			t.Fatal("got not production")
		}
	})

	panicTests := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "zero", value: "0"},
		{name: "true", value: "true"},
		{name: "false", value: "false"},
	}

	for _, test := range panicTests {
		t.Run("panics on "+test.name, func(t *testing.T) {
			t.Setenv("PRODUCTION", test.value)

			defer func() {
				if recover() == nil {
					t.Fatalf("expected a panic for %q", test.value)
				}
			}()

			IsProduction()
		})
	}
}

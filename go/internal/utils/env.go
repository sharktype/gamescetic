package utils

import (
	"fmt"
	"os"
)

func IsProduction() bool {
	production, isSet := os.LookupEnv("PRODUCTION")
	if !isSet {
		return false
	}

	if production != "1" {
		panic(fmt.Sprintf(`PRODUCTION must be unset or "1", got %q`, production))
	}

	return true
}

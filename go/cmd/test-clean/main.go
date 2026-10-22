package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sharktype/satchel/internal/db/dbtest"
)

func main() {
	if err := dbtest.Clean(context.Background()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "clean test databases:", err)
		
		os.Exit(1)
	}
}

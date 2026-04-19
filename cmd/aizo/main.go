package main

import (
	"fmt"
	"os"

	"github.com/realityos/aizo/cmd/aizo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

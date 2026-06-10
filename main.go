package main

import (
	"os"

	"github.com/ulgerang/gdc-sentinel/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

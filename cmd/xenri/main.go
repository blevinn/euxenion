// Command xenri is the operator-facing entry point for Euxenion.
package main

import (
	"os"

	"github.com/blevinn/euxenion/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}

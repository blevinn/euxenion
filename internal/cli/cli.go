// Package cli implements Xenri's command-line interface.
package cli

import (
	"fmt"
	"io"
)

// Version is overridden by the release build when versioned binaries exist.
var Version = "dev"

const usage = `Usage:
  xenri [command]

Commands:
  help      Show this help
  version   Print the xenri version

Euxenion is under development. MCP proxy operations are not implemented.
`

// Run executes a Xenri command and returns a process exit status.
// It does not exit the process, making it straightforward to test and embed.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "xenri: help does not accept arguments")
			return 2
		}
		fmt.Fprint(stdout, usage)
		return 0
	case "version", "--version":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "xenri: version does not accept arguments")
			return 2
		}
		fmt.Fprintf(stdout, "xenri %s\n", Version)
		return 0
	default:
		fmt.Fprintf(stderr, "xenri: unknown command %q\n", args[0])
		fmt.Fprint(stderr, usage)
		return 2
	}
}

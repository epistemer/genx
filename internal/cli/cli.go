// Package cli is genx's command line: it parses arguments and dispatches to
// the commands. cmd/genx and the end-to-end tests both call Main.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/epistemer/genx/internal/run"
)

const usage = `genx runs an app's code generation: producers, then trivial generators.

Usage:

	genx          run every step, write what changed, delete what nothing produces
	genx check    write nothing; fail if any output is stale, missing or left over
	genx list     show the steps in order, and what each reads and writes
	genx help     show this text
`

// Main runs genx with args, which exclude the program name, and returns the
// process exit code: 0 on success, 1 when a command fails, 2 on bad usage.
func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		dir, err := os.Getwd()
		if err == nil {
			err = run.Run(context.Background(), run.Options{Dir: dir, Stdout: stdout, Stderr: stderr})
		}
		if err != nil {
			fmt.Fprintf(stderr, "genx: %v\n", err)
			return 1
		}
		return 0
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	case "check", "list":
		return notYet(stderr, "genx "+args[0])
	default:
		fmt.Fprintf(stderr, "genx: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func notYet(stderr io.Writer, cmd string) int {
	fmt.Fprintf(stderr, "%s: not implemented yet\n", cmd)
	return 1
}

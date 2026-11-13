// Package cli is genx's command line: it parses arguments and dispatches to
// the commands. cmd/genx and the end-to-end tests both call Main.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

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
		return do(run.Run, stdout, stderr)
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	case "list":
		if len(args) > 1 {
			fmt.Fprintf(stderr, "genx list: takes no arguments\n\n%s", usage)
			return 2
		}
		return do(run.List, stdout, stderr)
	case "check":
		if len(args) > 1 {
			fmt.Fprintf(stderr, "genx check: takes no arguments\n\n%s", usage)
			return 2
		}
		return do(run.Check, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "genx: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// do runs a command in the working directory and turns its error into a
// message and exit code 1.
//
// The first interrupt (Ctrl-C) cancels the run: genx stops the step that is
// running, writes nothing more and reports it. A second one ends genx at once.
func do(cmd func(context.Context, run.Options) error, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()
	dir, err := os.Getwd()
	if err == nil {
		err = cmd(ctx, run.Options{Dir: dir, Stdout: stdout, Stderr: stderr})
	}
	if err != nil && ctx.Err() != nil && errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, "genx: interrupted; files already written stay, so run genx again")
		return 1
	}
	if err != nil {
		fmt.Fprintf(stderr, "genx: %v\n", err)
		return 1
	}
	return 0
}

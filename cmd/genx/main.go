// Command genx is neozp's code generation driver. See
// github.com/epistemer/genx for the manifest and metafile formats.
package main

import (
	"os"

	"github.com/epistemer/genx/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}

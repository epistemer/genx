// Package e2e runs genx's end-to-end testscript suites. Each script runs in a
// temporary module, example.com/app, with genx available as a command.
//
// Adapted from github.com/epistemer/cfgx/internal/e2e, which can't be
// imported (roadmap M0.3).
package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/epistemer/genx/internal/cli"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"genx":    func() { os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr)) },
		"produce": produce,
	})
}

// produce is a stand-in producer for scripts: produce src dst copies src to
// dst, as a program that writes a domain file would.
func produce() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: produce src dst")
		os.Exit(2)
	}
	b, err := os.ReadFile(os.Args[1])
	if err == nil {
		err = os.WriteFile(os.Args[2], b, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wrote", os.Args[2])
}

// goEnvKeys lists the Go environment variables copied into each script, so
// the go command there uses the host's warm caches and works offline.
var goEnvKeys = []string{
	"GOROOT", "GOCACHE", "GOMODCACHE", "GOPATH", "GOPROXY", "GOFLAGS",
	"GONOSUMDB", "GOPRIVATE", "GOINSECURE", "GONOPROXY", "GOSUMDB", "GOTOOLCHAIN",
}

// hostPassthroughEnvKeys lists host variables passed through when set, which
// the Go toolchain, the linker and crypto APIs need on Windows. PATH, TMP and
// TEMP are left to testscript: it puts genx first on PATH and gives each
// script its own temporary directory.
var hostPassthroughEnvKeys = []string{
	"USERPROFILE", "LOCALAPPDATA", "APPDATA", "SYSTEMROOT", "PATHEXT",
	"COMSPEC", "WINDIR",
}

func TestScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in -short mode")
	}
	goEnv := map[string]string{}
	if _, err := exec.LookPath("go"); err == nil {
		out, err := exec.Command("go", append([]string{"env", "-json"}, goEnvKeys...)...).Output()
		if err != nil {
			t.Fatalf("go env: %v", err)
		}
		if err := json.Unmarshal(out, &goEnv); err != nil {
			t.Fatalf("go env: %v", err)
		}
	}
	testscript.Run(t, testscript.Params{
		Dir:                 filepath.Join("testdata", "script"),
		RequireExplicitExec: true,
		Setup: func(e *testscript.Env) error {
			for _, k := range hostPassthroughEnvKeys {
				if v := os.Getenv(k); v != "" {
					e.Setenv(k, v)
				}
			}
			for _, k := range goEnvKeys {
				if v := goEnv[k]; v != "" {
					e.Setenv(k, v)
				}
			}
			// Every script is an app module unless it writes its own go.mod.
			gomod := filepath.Join(e.WorkDir, "go.mod")
			if _, err := os.Stat(gomod); err == nil {
				return nil
			}
			return os.WriteFile(gomod, []byte("module example.com/app\n\ngo 1.26.0\n"), 0o644)
		},
	})
}

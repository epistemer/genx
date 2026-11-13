// Package e2e runs genx's end-to-end testscript suites. Each script runs in a
// temporary module, example.com/app, with genx available as a command.
//
// Adapted from github.com/epistemer/cfgx/internal/e2e, which can't be
// imported (roadmap M0.3).
package e2e

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/goproxytest"
	"github.com/rogpeppe/go-internal/testscript"

	"github.com/epistemer/genx/internal/cli"
	"github.com/epistemer/genx/internal/golden"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"genx":    func() { os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr)) },
		"produce": produce,
		"consume": consume,
		"fail":    fail,
		"sleepy":  sleepy,
	})
}

// consume is a producer that reads a domain file: consume in out writes
// {"domain": "consumed/v1", "from": <in>} to out, and fails if in is missing.
func consume() {
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out := fmt.Sprintf("{\"domain\": \"consumed/v1\", \"from\": %s}\n", b)
	if err := os.WriteFile(os.Args[2], []byte(out), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// fail is a producer that fails: it prints its arguments to stderr.
func fail() {
	fmt.Fprintln(os.Stderr, "failing:", strings.Join(os.Args[1:], " "))
	os.Exit(3)
}

// sleepy is a producer that sleeps for a duration: sleepy 10s.
func sleepy() {
	d, _ := time.ParseDuration(os.Args[1])
	time.Sleep(d)
}

// produce is a stand-in producer for scripts: produce src dst copies src to
// dst, as a program that writes a domain file would.
func produce() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: produce src dst")
		os.Exit(2)
	}
	// Under go generate, genx must not pass on the variables that describe
	// its own directive (interface amendment A2).
	for _, v := range []string{"GOFILE", "GOLINE", "GOPACKAGE", "DOLLAR"} {
		if x, ok := os.LookupEnv(v); ok {
			fmt.Fprintf(os.Stderr, "produce: %s is set to %q\n", v, x)
			os.Exit(1)
		}
	}
	b, err := os.ReadFile(os.Args[1])
	if err == nil {
		err = os.MkdirAll(filepath.Dir(os.Args[2]), 0o755)
	}
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
	modDir, err := filepath.Abs(filepath.Join("testdata", "mod"))
	if err != nil {
		t.Fatal(err)
	}
	srv, err := goproxytest.NewServer(modDir, "")
	if err != nil {
		t.Fatalf("goproxytest.NewServer: %v", err)
	}
	t.Cleanup(srv.Close)

	testscript.Run(t, testscript.Params{
		Dir:                 filepath.Join("testdata", "script"),
		RequireExplicitExec: true,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"treehash": cmdTreeHash,
		},
		// go test ./internal/e2e -update rewrites the files scripts cmp
		// against; read the diff before committing it.
		UpdateScripts: golden.Update(),
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
			e.Setenv("GOPROXY", srv.URL)
			e.Setenv("GONOPROXY", "")
			e.Setenv("GOSUMDB", "off")
			e.Setenv("GONOSUMDB", "*")
			e.Setenv("GONOSUMCHECK", "*")

			// Every script is an app module unless it writes its own go.mod.
			gomod := filepath.Join(e.WorkDir, "go.mod")
			if _, err := os.Stat(gomod); err == nil {
				return nil
			}
			return os.WriteFile(gomod, []byte("module example.com/app\n\ngo 1.26.0\n"), 0o644)
		},
	})
}

// cmdTreeHash prints one hash of every file under the work directory, with
// its path, so a script can show a command wrote nothing. Files named
// treehash-* are left out, so a script can keep the hash in one:
// treehash; cp stdout treehash-before.
func cmdTreeHash(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 0 {
		ts.Fatalf("usage: treehash")
	}
	root := ts.MkAbs(".")
	h := sha256.New()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), "treehash-") {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s %x\n", filepath.ToSlash(rel), sha256.Sum256(b))
		return nil
	})
	ts.Check(err)
	fmt.Fprintf(ts.Stdout(), "%x\n", h.Sum(nil))
}

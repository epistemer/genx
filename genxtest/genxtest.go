package genxtest

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/epistemer/genx/internal/run"
)

// AssertGenerated fails t when `genx check` would fail in the module that
// holds the test's working directory: an output is stale, missing or
// leftover, the lock would change, or a producer's check command fails. It
// writes nothing. Producers without a check command are trusted, as under
// genx check.
func AssertGenerated(t testing.TB) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	err = run.Check(context.Background(), run.Options{Dir: dir, Stdout: &out, Stderr: &out})
	if err != nil {
		// What the producers' check commands printed explains the failure.
		if s := strings.TrimSpace(out.String()); s != "" {
			t.Fatalf("genx check: %v\n%s", err, s)
		}
		t.Fatalf("genx check: %v", err)
	}
}

// Input is a domain file for Render. Domain is empty unless the file is
// foreign and doesn't name its domain itself.
type Input struct {
	Path   string
	Domain string
}

// Render runs one generator of the metafile in dir on the given domain
// files, with no manifest and no producers, and returns the files it would
// write, keyed by slash-separated path. It writes nothing, and fails t on
// any error the generator would make genx report.
// Every parameter the metafile declares keeps its default.
func Render(t testing.TB, dir, generator string, inputs ...Input) map[string][]byte {
	t.Helper()
	return render(t, "genxtest.Render", dir, generator, nil, inputs)
}

// RenderParams is Render with parameters set, as an app's use entry sets
// them (interface amendment A4); the others keep their defaults. A value is
// read as its JSON encoding. A parameter the metafile doesn't declare, or a
// value whose JSON type differs from a non-null default's, fails t, as it
// fails genx.
func RenderParams(t testing.TB, dir, generator string, params map[string]any, inputs ...Input) map[string][]byte {
	t.Helper()
	return render(t, "genxtest.RenderParams", dir, generator, params, inputs)
}

func render(t testing.TB, fn, dir, generator string, params map[string]any, inputs []Input) map[string][]byte {
	t.Helper()
	in := make([]run.Input, len(inputs))
	for i, x := range inputs {
		in[i] = run.Input{Path: x.Path, Domain: x.Domain}
	}
	files, err := run.Render(context.Background(), dir, generator, params, in)
	if err != nil {
		t.Fatalf("%s: %v", fn, err)
	}
	return files
}

package genxtest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/epistemer/genx/genxtest"
	"github.com/epistemer/genx/internal/run"
)

// recorder stands in for t where a failure is expected: Fatalf records the
// message and ends the goroutine, as testing.T's does.
type recorder struct {
	testing.TB
	msg string
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.msg = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func (r *recorder) Fatal(args ...any) {
	r.msg = fmt.Sprint(args...)
	runtime.Goexit()
}

// fails runs f with a recorder and returns what it failed with, or "".
func fails(t *testing.T, f func(testing.TB)) string {
	r := &recorder{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(r)
	}()
	<-done
	return r.msg
}

func TestRender(t *testing.T) {
	// Inputs are read in path order, whatever order they're given in.
	files := genxtest.Render(t, "testdata/gen", "list",
		genxtest.Input{Path: "testdata/in/b.labels.json"},
		genxtest.Input{Path: "testdata/in/a.labels.json"})
	if want := map[string][]byte{"zz_list.txt": []byte("a\nb\n")}; !reflect.DeepEqual(files, want) {
		t.Errorf("list: got %q, want %q", files, want)
	}

	// A foreign file gets its domain from the Input.
	files = genxtest.Render(t, "testdata/gen", "config",
		genxtest.Input{Path: "testdata/in/config.yaml", Domain: "app.config/v1"})
	if got := string(files["zz_shop.txt"]); got != "name: shop\n" {
		t.Errorf("config: got %q", got)
	}

	// A generator that emits a domain returns the file genx would write.
	files = genxtest.Render(t, "testdata/gen", "emit",
		genxtest.Input{Path: "testdata/in/config.yaml", Domain: "app.config/v1"})
	want := "{\n  \"domain\": \"labels/v1\",\n  \"labels\": [],\n  \"package\": {\n    \"dir\": \".\",\n    \"name\": \"app\"\n  },\n  \"producer\": \"shop\"\n}\n"
	if got := string(files["zz_app.labels.json"]); got != want {
		t.Errorf("emit: got %q, want %q", got, want)
	}
}

func TestRenderFails(t *testing.T) {
	for _, tt := range []struct {
		name, generator string
		inputs          []genxtest.Input
		want            string
	}{
		{"out leaves the module", "badout", []genxtest.Input{{Path: "testdata/in/a.labels.json"}},
			`genxtest.Render: testdata/gen:badout: out: "../zz_x.txt": must not contain ".."`},
		{"another major version", "list", []genxtest.Input{{Path: "testdata/in/v2.labels.json"}},
			"genxtest.Render: testdata/gen:list needs labels/v1, but testdata/in/v2.labels.json says labels/v2"},
		{"a foreign file without a Domain", "config", []genxtest.Input{{Path: "testdata/in/config.yaml"}},
			`genxtest.Render: testdata/in/config.yaml: no "domain" field (give a foreign file's Input a Domain)`},
		{"no such generator", "nope", nil,
			`genxtest.Render: use testdata/gen: no generator "nope" in its genx.meta.yaml (it has: list, config, emit, badout)`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := fails(t, func(tb testing.TB) { genxtest.Render(tb, "testdata/gen", tt.generator, tt.inputs...) })
			if got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

// module writes a module that runs the list generator over one labels file
// a trusted producer declares, and returns its directory.
func module(t *testing.T) string {
	dir := t.TempDir()
	write := func(name, content string) {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/m\n\ngo 1.26\n")
	write("genx.yaml", "version: 1\nproducers:\n  - name: a\n    run: [go, version]\n    emits: [a/zz_a.labels.json]\nuse:\n  - example.com/m/gen: [list]\n")
	for _, f := range []string{"genx.meta.yaml", "list.tmpl"} {
		b, err := os.ReadFile(filepath.Join("testdata", "gen", f))
		if err != nil {
			t.Fatal(err)
		}
		write("gen/"+f, string(b))
	}
	write("gen/gen.go", "package gen\n")
	write("a/zz_a.labels.json", `{"domain": "labels/v1", "producer": "a", "package": {"name": "a", "dir": "a"}, "labels": []}`+"\n")
	if err := run.Run(t.Context(), run.Options{Dir: dir, Stdout: &strings.Builder{}, Stderr: &strings.Builder{}}); err != nil {
		t.Fatal(err)
	}
	return dir
}

// snapshot is every file under dir with its content.
func snapshot(t *testing.T, dir string) map[string]string {
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		files[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestAssertGenerated(t *testing.T) {
	dir := module(t)
	t.Chdir(filepath.Join(dir, "gen")) // any directory in the module

	before := snapshot(t, dir)
	genxtest.AssertGenerated(t)
	if after := snapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Errorf("AssertGenerated changed the module:\nbefore %q\nafter  %q", before, after)
	}

	if err := os.WriteFile(filepath.Join(dir, "zz_list.txt"), []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := fails(t, genxtest.AssertGenerated)
	if want := "genx check: check found 1 problem; run genx:\n  stale     zz_list.txt (example.com/m/gen:list)\na: not checked (no check command)"; got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestRenderParams(t *testing.T) {
	in := genxtest.Input{Path: "testdata/in/a.labels.json"}

	// Render keeps every default.
	files := genxtest.Render(t, "testdata/params", "show", in)
	if want := map[string][]byte{"models/zz_models.txt": []byte("package models: a from a \n")}; !reflect.DeepEqual(files, want) {
		t.Errorf("defaults: got %q, want %q", files, want)
	}

	// RenderParams sets some; the others keep their defaults. Go values are
	// read as their JSON encoding, so a []string is an array.
	files = genxtest.RenderParams(t, "testdata/params", "show",
		map[string]any{"dir": "internal/db", "tags": []string{"x", "y"}}, in)
	if want := map[string][]byte{"internal/db/zz_db.txt": []byte("package db: x,y from a \n")}; !reflect.DeepEqual(files, want) {
		t.Errorf("params: got %q, want %q", files, want)
	}

	// A null default accepts any type.
	files = genxtest.RenderParams(t, "testdata/params", "show", map[string]any{"package": "store"}, in)
	if _, ok := files["models/zz_store.txt"]; !ok {
		t.Errorf("package: got %q", files)
	}

	// The same params give the same bytes.
	again := genxtest.RenderParams(t, "testdata/params", "show",
		map[string]any{"dir": "internal/db", "tags": []string{"x", "y"}}, in)
	if !reflect.DeepEqual(again, genxtest.RenderParams(t, "testdata/params", "show",
		map[string]any{"tags": []string{"x", "y"}, "dir": "internal/db"}, in)) {
		t.Error("two renders with the same params differ")
	}
}

func TestRenderParamsFails(t *testing.T) {
	in := genxtest.Input{Path: "testdata/in/a.labels.json"}
	for _, tt := range []struct {
		name   string
		dir    string
		params map[string]any
		want   string
	}{
		{"an unknown parameter", "testdata/params", map[string]any{"colour": "blue"},
			`genxtest.RenderParams: params.colour: testdata/params/genx.meta.yaml declares no parameter "colour" (it declares: dir, package, tags)`},
		{"no parameters declared", "testdata/gen", map[string]any{"dir": "db"},
			`genxtest.RenderParams: params.dir: testdata/gen/genx.meta.yaml declares no parameter "dir" (it declares none)`},
		{"a wrong type", "testdata/params", map[string]any{"dir": 3},
			`genxtest.RenderParams: params.dir: must be a string, as its default "models" is (got a number)`},
		{"null for a string", "testdata/params", map[string]any{"dir": nil},
			`genxtest.RenderParams: params.dir: must be a string, as its default "models" is (got null)`},
		{"two problems at once", "testdata/params", map[string]any{"tags": "x", "colour": 1},
			"genxtest.RenderParams: params.colour: testdata/params/genx.meta.yaml declares no parameter \"colour\" (it declares: dir, package, tags)\n" +
				`params.tags: must be an array, as its default ["a"] is (got a string)`},
		{"a value JSON can't encode", "testdata/params", map[string]any{"dir": func() {}},
			"genxtest.RenderParams: params.dir: json: unsupported type: func()"},
		{"every problem, in name order", "testdata/params", map[string]any{"tags": func() {}, "dir": make(chan int), "colour": 1},
			"genxtest.RenderParams: params.dir: json: unsupported type: chan int\n" +
				"params.tags: json: unsupported type: func()\n" +
				`params.colour: testdata/params/genx.meta.yaml declares no parameter "colour" (it declares: dir, package, tags)`},
		{"a query refuses a value", "testdata/params", map[string]any{"dir": "../x"},
			`genxtest.RenderParams: testdata/params:show: out: "../x/zz_x.txt": must not contain ".."`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gen := "show"
			if tt.dir == "testdata/gen" {
				gen = "list"
			}
			got := fails(t, func(tb testing.TB) { genxtest.RenderParams(tb, tt.dir, gen, tt.params, in) })
			if got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

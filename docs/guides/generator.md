# Writing a trivial generator

A trivial generator turns domain files into code with a jq query and a Go
template, and nothing else: no Go code of yours runs. It ships in a Go
package as `genx.meta.yaml` and its templates, and an app switches it on
with `use`. This guide writes the generator of
[examples/labels](../../examples/labels), which CI runs.

The contract is [interface.md](../interface.md), §4–§7.

## 1. The package

The metafile and its templates live in the directory of an importable Go
package, so that `go mod vendor` copies them (ADR 0003). The package needs
one Go file, which can be empty but for its doc comment.

<!-- file: gen/gen.go -->
```go
// Package gen ships the example's trivial generator, labels, which turns
// every labels/v1 file into Go: L10n, which gives a label's text by key,
// and a constant for each label's key that has a var. It holds no code: genx.meta.yaml and labels.go.tmpl are all
// of it. The Go file is here so that the package can be imported, which
// keeps it in go.mod and in vendor/ (genx ADR 0003).
package gen
```

## 2. The metafile

<!-- file: gen/genx.meta.yaml -->
```yaml
version: 1

generators:
  labels:
    doc: the texts of every label by key, and a constant per key
    needs: [labels/v1]
    # One unit per package: the labels of all its files, one per key. The
    # template declares only each var, L10n and l10n, as labels/v1 allows.
    units: |
      .["labels/v1"] | group_by(.package.dir)[]
      | .[0].package as $package
      | [.[].labels[] | {key, text, var: (.var // null)}]
      | (group_by(.key) | map(select(map(.text) | unique | length > 1) | .[0].key)) as $clash
      | if $clash != [] then error("labels: one key, two texts: \($clash | join(", "))") else . end
      | {package: $package, labels: unique_by(.key)}
    out: '"\(.package.dir)/zz_\(.package.name)_labels.go"'
    template: labels.go.tmpl
```

- `needs` lists the domains it reads. The input is an object keyed by
  domain, each holding that domain's files in path order.
- `units` is a jq query. Each value it outputs is rendered into one file.
  Here, one unit per Go package, with the labels of every file for that
  package, merged.
- `out` is a query over a unit: the file's path, relative to the module
  root. Its base name starts with `zz_`.
- Give every key the template reads a value, `null` if need be
  (`.var // null` above): the template stops on a missing key, which also
  catches typos.
- Queries can't read the clock, the environment or other files (§5). Use
  `\p{L}` and `\p{N}` in regular expressions, not `[[:alnum:]]`.
- To write a domain instead of code, replace `template` with `emits` and
  make each unit the domain document; genx encodes it.

## 3. The template

<!-- file: gen/labels.go.tmpl -->
```gotemplate
// Code generated from labels by github.com/epistemer/genx/examples/labels/gen. DO NOT EDIT.

package {{.package.name}}

// l10n holds every label's text in the source language, by key.
var l10n = map[string]string{
{{- range .labels}}
	{{printf "%q" .key}}: {{printf "%q" .text}},
{{- end}}
}

// L10n returns the text of the label with the key, or "" if there is none.
func L10n(key string) string { return l10n[key] }

// The keys of the labels that have a name.
const (
{{- range .labels}}{{if .var}}
	{{.var}} = {{printf "%q" .key}}
{{- end}}{{end}}
)
```

- Start a Go file with the generated-code header; genx refuses one without
  it, and uses it to tell generated files from edited ones.
- `text/template` with no added functions. genx formats `.go` output with
  gofmt, so don't fuss over spacing.
- Several producers can feed one package, so declare nothing but what the
  domain lets you: for labels, each `var`, `L10n` and `l10n` (§11). The
  example's `l10n` map would not compile with one key twice, so its query
  keeps one label per key and refuses one key with two texts.

## 4. Test it without an app

`genxtest.Render` runs one generator on example files, with no manifest
and no producers, and returns what it would write. Compare it with a golden
file.

<!-- file: gen/gen_test.go -->
```go
package gen

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/epistemer/genx/genxtest"
)

var update = flag.Bool("update", false, "rewrite the golden file")

// TestLabels renders the generator on a labels file, with no manifest, and
// compares the result with a golden file.
func TestLabels(t *testing.T) {
	files := genxtest.Render(t, ".", "labels", genxtest.Input{Path: "testdata/settings.labels.json"})
	got, ok := files["internal/settings/zz_settings_labels.go"]
	if !ok || len(files) != 1 {
		t.Fatalf("got files %v, want internal/settings/zz_settings_labels.go only", keys(files))
	}
	const golden = "testdata/zz_settings_labels.go.golden"
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func keys(m map[string][]byte) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
```

## 5. Switch it on in an app

```yaml
use:
  - github.com/epistemer/genx/examples/labels/gen: [labels]
```

`genx list` shows it in the app's order; generators in the build that the
app hasn't switched on are listed after the steps.

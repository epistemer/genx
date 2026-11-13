# Writing a producer

A producer is any program that writes domain files: cfgx, sqlc, or a
command in your own module. genx runs it, reads what it declared, and hands
the data to the generators that need it. This guide writes one in Go that
publishes `labels/v1`, the domain i18n reads. The whole example is
[examples/labels](../../examples/labels), and CI runs it.

The contract is [interface.md](../interface.md): §3 for the manifest, §10
for domains, §11 for `labels/v1`.

## 1. Decide what you publish

Publish facts that are already resolved: the Go names, the package, the
paths, after every naming rule your program applies. A consumer never works
a name out again (ADR 0001 §3). For labels, that is each label's key, its
text in the source language, and, if you want a Go handle for it, its `var`.

Name the file `zz_<package>.<domain name>.json`, next to the code it
describes, and commit it.

## 2. Write the file

The example keeps its settings in a table; a real producer reads a schema.
It writes the file only from what it was given, so two runs give the same
bytes.

<!-- file: cmd/settings/main.go -->
```go
// Command settings is the example's producer: it writes the labels of the
// app's settings as a labels/v1 file. A real producer, such as cfgx, reads
// them from a schema; this one keeps them in a table.
//
// With -check it writes nothing, and fails if the file isn't what it would
// write, so genx check can run it.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// out is where the labels go, relative to the module root, where genx runs
// this producer.
const out = "internal/settings/zz_settings.labels.json"

// setting is one setting the app has.
type setting struct {
	path, text, name string
}

var settings = []setting{
	{"server.port", "Listen port", "Port"},
	{"server.host", "Listen address", "Host"},
	{"log.level", "Minimum log level", "LogLevel"},
}

func main() {
	check := flag.Bool("check", false, "write nothing; fail if the file is stale")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, "settings:", err)
		os.Exit(1)
	}
}

func run(check bool) error {
	labels := []map[string]string{}
	for _, s := range settings {
		labels = append(labels, map[string]string{
			"key":  "settings." + s.path,
			"text": s.text,
			"path": s.path,
			"var":  "L10n" + s.name,
		})
	}
	doc := map[string]any{
		"domain":   "labels/v1",
		"producer": "settings",
		"package":  map[string]string{"name": "settings", "dir": "internal/settings"},
		"labels":   labels,
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if check {
		old, err := os.ReadFile(out)
		if err != nil || !bytes.Equal(old, b) {
			return fmt.Errorf("%s is stale; run genx", out)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, b, 0o644)
}
```

Things to get right:

- **Encode deterministically.** `encoding/json` sorts map keys. Keep
  `labels` in a stable order: consumers keep it.
- **Check your own output** against the conformance files in
  [testdata/labels](../../testdata/labels): every valid file's rules apply
  to yours, and genx refuses a file that breaks one, naming the field.
- **Keep the reserved names free.** A renderer may declare each `var`,
  `L10n` and `l10n` in your package (§11).

## 3. Give it a check mode

`genx check` runs your producer's `check` command, which must write
nothing and exit non-zero when the file is stale. Without one, genx trusts
the file and says so on every check (ADR 0007). The example's `-check`
compares what it would write with what is there.

## 4. Declare it in the manifest

<!-- file: genx.yaml -->
```yaml
version: 1

producers:
  - name: settings
    run: [go, run, ./cmd/settings]
    check: [go, run, ./cmd/settings, -check]
    emits:
      - internal/settings/zz_settings.labels.json

use:
  - github.com/epistemer/genx/examples/labels/gen: [labels]
```

`run` and `check` are argv lists, run without a shell from the module root
(or `dir`). `emits` lists every file the producer writes that a generator
may read. A file that doesn't say its domain, such as sqlc's request, gets
one here: `{path: models/codegen_request.json, domain: sqlc.request/v1}`.

## 5. Test it

`genxtest.AssertGenerated` fails a test when `genx check` would: it runs
your check command and every generator, and writes nothing.

<!-- file: example_test.go -->
```go
package main

import (
	"testing"

	"github.com/epistemer/genx/genxtest"
)

// TestGenerated fails when a generated file is stale, as genx check does.
func TestGenerated(t *testing.T) {
	genxtest.AssertGenerated(t)
}
```

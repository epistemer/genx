# genx — Interface

> Status: **locked by epistemer on 2026-11-13** (M1.6), after
> [review 0003](reviews/0003-gemini-interface-review.md). epistemer chose:
> `use` names packages, no `vars`, and `zz_` names for generator outputs.
> A change to anything here is an amendment, listed in §13 with its reason.
>
> This document is the contract: what an app author and a package author
> write, and what genx does with it. [ADR 0001](adr/0001-generation-pipeline.md)
> says why; ADRs 0003–0008 hold the evidence for each choice.

## 1. Principles

1. A producer never names a consumer; a consumer never names a producer. The
   app's manifest is the only file that names both.
2. Nothing runs unless the manifest names it. Adding a dependency never adds
   a generated file.
3. A trivial generator is data: queries and a template. genx runs no code
   from dependencies, and a query sees only its input.
4. The same inputs give the same bytes, on every OS.
5. `genx check` writes nothing.

## 2. Files

| File | Where | Written by |
|------|-------|-----------|
| `genx.yaml`, the manifest | The app's module root, next to `go.mod` | The app's author |
| `genx.meta.yaml`, a metafile, and its templates | The directory of an importable Go package (not `main`) | The package's author |
| `genx.lock`, the list of files genx wrote | The module root | genx; committed |
| Domain files | Next to the code they describe (§10) | Producers, and generators that emit a domain |

- **The module root** is the directory of the nearest `go.mod` above the
  working directory. genx works on that module only; under `go.work`, each
  module that generates has its own manifest. A missing `genx.yaml` there is
  an error that names the directory.
- genx reads files in the module through `os.Root` at the module root, and a
  metafile's templates through `os.Root` at the metafile's directory, so a
  symlink can't lead outside either.
- **Both YAML files are strict:** one document, unknown fields and duplicate
  keys are errors, and messages carry the file and line. A `version` above 1
  says to upgrade genx.
- genx writes LF line endings in every file it writes.

## 3. The manifest, `genx.yaml`

```yaml
version: 1

producers:
  - name: settings
    dir: internal/settings
    run:   [go, tool, cfgx, gen, ./schema]
    check: [go, tool, cfgx, gen, -check, ./schema]
    emits:
      - internal/settings/zz_settings.labels.json

  - name: models
    run: [go, tool, sqlc, generate, -f, config/sqlc.yaml]
    check: [go, tool, sqlc, diff, -f, config/sqlc.yaml]
    emits:
      - path: models/codegen_request.json
        domain: sqlc.request/v1

use:
  - github.com/epistemer/i18n: [labels]
  - example.com/app/internal/gen: [enum-labels]
```

| Field | Required | Meaning |
|-------|----------|---------|
| `version` | yes | `1` |
| `producers[].name` | yes | Unique; `[a-z][a-z0-9-]*`. Used in messages and as the prefix of the producer's output |
| `producers[].dir` | no | Where the commands run: a path under §7's rules for elements, relative to the module root. Default `.` |
| `producers[].run` | yes | The command, as a non-empty argv. No shell |
| `producers[].check` | no | A non-empty argv that writes nothing and exits non-zero when the producer's output is stale (§9) |
| `producers[].timeout` | no | A positive Go duration. Default `10m` |
| `producers[].needs` | no | Domains the producer reads; it runs after the steps that write them |
| `producers[].emits` | yes | The domain files it writes, relative to the module root. A string, or `{path, domain}` for a foreign file, which doesn't name its domain itself |
| `use` | no | A list of one-key maps: a Go **package import path** → the generators of that package's metafile to switch on, in order |

- `use` names packages (ADR 0003). genx resolves them all with one
  `go list -f '{{.Dir}}'`, so module mode, `replace`, `go.work` and
  `-mod=vendor` work as the go command does. The app's own packages work too.
- A package named in `use` must be one the go command can list. When `go mod
  tidy` removed it because nothing imports it yet, or `go mod vendor` didn't
  copy it, the message says so and shows the fix: a file the app keeps with a
  build-tagged blank import (ADR 0003).
- One generator appears once in `use`; a repeat is an error.

## 4. The metafile, `genx.meta.yaml`

```yaml
version: 1

defs: |
  def pascal: [splits("[^\\p{L}\\p{N}]+") | select(length > 0)
               | (.[0:1] | ascii_upcase) + .[1:]] | join("");

generators:
  labels:
    doc: i18n keys for every label, one file per package
    needs: [labels/v1]
    units: |
      .["labels/v1"] | group_by(.package.dir)[]
      | {package: .[0].package, labels: [.[].labels[]]}
    out: '"\(.package.dir)/zz_\(.package.name)_i18n.go"'
    template: labels.go.tmpl

  enum-labels:
    doc: a labels file for the enums in sqlc's request
    needs: [sqlc.request/v1]
    emits: labels/v1
    units: |   # abridged
      .["sqlc.request/v1"][0]
      | {domain: "labels/v1", producer: "models",
         package: {name: "models", dir: "models"},
         labels: [.catalog.schemas[].enums[] | ...]}
    out: '"models/zz_enums.labels.json"'
```

| Field | Required | Meaning |
|-------|----------|---------|
| `version` | yes | `1` |
| `defs` | no | jq definitions, put before every query of this metafile, followed by a newline |
| `generators.<name>` | – | `[a-z][a-z0-9-]*`, unique in the metafile |
| `doc` | yes | One line, shown by `genx list` |
| `needs` | yes | The domains it reads |
| `units` | yes | A query over the input (§5). Each value it outputs is one unit, rendered into one file. No output means no files |
| `out` | yes | A query over a unit. It must output exactly one string: the file's path, cleaned and then checked (§7) |
| `template` | one of the two | A file name, with no `/`, in the metafile's directory. The unit is the template's data (`.`) |
| `emits` | one of the two | The one domain this generator writes, as a string. genx writes each unit as JSON (§10); the unit's `domain` field must name this domain |

- A unit is any JSON value, usually an object. There is no separate list of
  template variables (ADR 0005).
- A generator with `emits` has no template: a domain file is the unit,
  encoded by genx (ADR 0001, change 11).
- Templates sit next to the metafile because `go mod vendor` copies only the
  files of a package directory (ADR 0003).

## 5. What a query reads and does

- **Input.** An object keyed by each domain in `needs`. Each holds the
  domain's documents as an array, in the byte order of their slash-separated
  paths; an empty array when there are none.
- **Documents.** One JSON value per file, or one YAML document when the
  extension is `.yaml` or `.yml`. Integers keep their precision (ADR 0004).
  A YAML file is read as its JSON equivalent: keys must be strings; a
  timestamp stays the string written; `.nan` and `.inf` are refused; aliases
  are expanded. Object keys are seen in sorted order; arrays keep theirs.
- **Language.** jq as gojq v0.12.20 implements it. These are refused when
  the metafile is read, with the name and the generator: `now`, `mktime`,
  `gmtime`, `localtime`, `strftime`, `strflocaltime`, `strptime`, `todate`,
  `fromdate`, `date`, `dateadd`, `datesub`, `todateiso8601`,
  `fromdateiso8601`, `env`, `$ENV`, `input`, `inputs`, `input_filename`,
  `input_line_number`, `debug`, `stderr`, `halt`, `halt_error`, `import`,
  `include`, `get_search_list`, `modulemeta`, `$__loc__`.
- **Failing.** `error("...")` stops the run. genx prints the message with the
  generator, its package and the input files.
- **Time.** A generator's queries share 30 seconds, checked between units.
  A template can't be stopped once it runs.

## 6. Templates

- Go's `text/template` with no added functions, and `missingkey=error`. A
  key the template reads must be in the unit, `null` if need be: an absent
  key stops the render, which also catches typos.
- Read as UTF-8 with CRLF turned into LF.
- A `.go` output goes through `go/format`; a failure shows the generated
  lines around the error.
- After formatting, a `.go` output must begin with a line matching Go's
  convention, `^// Code generated .* DO NOT EDIT\.$`. The template writes
  it; genx refuses an output without it.
- **An empty render** (only white space) writes no file. A file from an
  earlier run is then leftover, and is deleted (§8).

## 7. Output paths

The paths genx writes (`out`) and the paths producers declare (`emits`):

- are relative to the module root, slash-separated, and equal to their
  `path.Clean` form, which isn't `.`; `out` is cleaned first, so
  `./zz_a.go` is `zz_a.go`;
- have elements made only of `A–Z a–z 0–9 . _ -`, that don't start with `.`
  or end with `.`, and whose part before the first `.` isn't a Windows
  reserved name (`CON`, `PRN`, `AUX`, `NUL`, `COM1`–`COM9`, `LPT1`–`LPT9`), in
  any case;
- belong to one step only. Paths are compared without case (Unicode simple
  folding), across every step.

A path from `out` also has a base name that starts with `zz_`. A producer's
paths don't: a foreign file keeps its tool's name.

genx writes and deletes through `os.Root` at the module root. It writes a
file only when its bytes change, through a temporary file in the same
directory and a rename, and removes the temporary file on failure. Every
unit renders, and every rule here and in §8 is checked, before any file is
written or deleted.

## 8. The lock, `genx.lock`

```
# Code generated by genx. DO NOT EDIT.
# The files genx wrote, and the generator that wrote each.
internal/settings/zz_settings_i18n.go github.com/epistemer/i18n:labels
models/zz_enums.labels.json example.com/app/internal/gen:enum-labels
```

- One line per file a **generator** wrote: the path, a space, and
  `<package>:<generator>`. Sorted by path in byte order. No hashes, no times,
  so two runs give the same lock. Paths have no spaces (§7). A line genx
  can't parse is an error.
- Producers' files aren't in it: a producer cleans up after itself.
- **Leftover output** is a file in the previous lock that this run doesn't
  produce. genx deletes it when it still looks generated: a `.go` file whose
  first line is the header (§6, CR ignored); any other file, by being in the
  lock. A `.go` file without its header stops the run, named, and is kept. A
  leftover already gone from the disk is dropped from the lock.
- **Never replace.** An output may not overwrite an existing file unless
  the lock lists it, compared without case, and a `.go` file in the lock
  must still have its header. Leftovers are deleted before outputs are
  written, so a rename that only changes case works.

## 9. Commands

| Command | Does | Exit |
|---------|------|------|
| `genx` | Runs every step in order, deletes leftovers, writes what changed, writes the lock | 0, or 1 on a failure |
| `genx check` | Writes nothing (below) | 0 if all is current, 1 otherwise |
| `genx list` | The steps in order, with what each reads and writes; then the generators of other packages in the build (`go list all`) that `use` doesn't name. Runs no producer and no query | 0, or 1 on a failure |
| `genx help` | Usage | 0 |

- Usage errors exit 2.
- **Order.** A step that needs a domain runs after every step that writes it.
  Among the steps ready to run, the one first in the manifest goes first:
  producers in their order, then generators in `use` order. A domain that a
  step needs and no step writes, a cycle, or two steps writing one file is
  an error.
- **Producers** run with the environment genx has, less the variables
  `go generate` sets for its directive (A2), in their `dir`, with their
  output prefixed by `<name>: `. After `run`, and under `check`, each
  declared file must exist, hold one JSON or YAML value, and, unless it's a
  foreign file, name its domain in a `domain` field.
- **`genx check`** reports every problem it finds, not just the first:
  - a producer with `check` runs it; a non-zero exit fails `check`. A
    producer without one is trusted, and `check` prints
    `<name>: not checked (no check command)` (ADR 0007);
  - every generator renders in memory, each over what the earlier ones
    rendered. An output whose bytes differ is **stale**; one that should
    exist and doesn't is **missing**; a file in the lock that nothing
    produces is **leftover**; a lock that differs from what `genx` would
    write fails too.
- **`genx list`** shows a package whose metafile doesn't parse with a
  warning, and still exits 0; a metafile named in `use` must parse.
- **`go generate`.** Put `//go:generate go tool genx` in a file of the root
  package, or of any package genx doesn't write into: `go generate` lists a
  package's files before it runs.

## 10. Domains

- A domain is `name/vN`. `name` is one or more parts matching `[a-z][a-z0-9-]*`
  joined by `.`; `N` matches `[1-9][0-9]*`. Within a major version a domain
  only gains optional fields, so a reader ignores fields it doesn't know.
- A domain file says its domain in a top-level `domain` field, unless it's a
  foreign file named in the manifest.
- Domain files are committed. A neozp producer names them
  `zz_<name>.<domain name>.json`, such as `zz_settings.labels.json`; a
  foreign file keeps its tool's name.
- genx writes a domain file (from `emits`) as JSON with keys sorted, two
  spaces of indentation, integers as written, and a final newline.
- A file whose domain has the name of one a step needs but another major
  version stops the run, naming the file, the step, and both versions. A
  domain nobody needs is fine.

## 11. `labels/v1`

ADR 0001 §4:

| Field | Required | Meaning |
|-------|----------|---------|
| `domain` | yes | `labels/v1` |
| `producer` | yes | Who wrote the file; used in messages |
| `package.name` | yes | The Go package the labels belong to |
| `package.dir` | yes | Its directory, relative to the module root; `.` for the root package |
| `labels[].key` | yes | The text's name across the app. One key has one text |
| `labels[].text` | yes | The text in the source language. It may be empty |
| `labels[].path` | no | A name for finding the label at run time in its package; unique there |
| `labels[].var` | no | A Go identifier for the label's handle; unique in the package |

- `labels` keeps its order.
- A renderer that writes Go into `package` may declare each `var`, `L10n`
  and `l10n`, and nothing else at package level.
- The JSON Schema is [schema/labels.v1.json](../schema/labels.v1.json). It
  checks the shape. genx checks each labels file it reads beyond it:
  `package.name` and every `var` are Go identifiers, not keywords and not
  `_`; no `var` is `L10n` or `l10n`; `package.dir` is `.` or follows §7's
  element rules; and `var` and `path` are each unique in the file.
- Checks across files are the consumer's: i18n's generator refuses one key
  with two texts, and a `var` or `path` used twice in a package.

## 12. Not in this version

Parameters for a generator in `use`; `meta` on labels; producers described
by their modules; `genx init` and `genx explain`; parallel steps and caching;
a watch mode. The roadmap's backlog orders them.

## 13. Amendments

### A1. `genxtest` (2026-11-13, M7.3, decided by epistemer)

§12 left `genxtest`'s API for review. epistemer kept it as proposed:

```go
// AssertGenerated fails t when genx check would fail in the module that
// holds the test's working directory. It writes nothing.
func AssertGenerated(t testing.TB)

// Input is a domain file for Render; Domain only for a foreign file.
type Input struct{ Path, Domain string }

// Render runs one generator of the metafile in dir on the inputs, with no
// manifest and no producers, and returns the files it would write by
// slash-separated path. Any error genx would report fails t.
func Render(t testing.TB, dir, generator string, inputs ...Input) map[string][]byte
```

A way to test a generator's failures (`RenderErr`, from review 0008) was
considered and left out.

### A2. Producers don't see go generate's variables (2026-11-13, M8.4)

§9 says producers run with the environment genx has. Under `go generate`
that includes `GOFILE`, `GOLINE`, `GOPACKAGE` and `DOLLAR`, which describe
genx's directive. A producer that reads them writes into the wrong package:
cfgx, run for `internal/settings` from a directive in `package main`, wrote
`zz_main.go`. genx now runs producers without those four variables. A
clarification: a producer was never meant to see another directive's
location.

# What can fail, and what genx says

Every failure genx reports, with an excerpt of its message and the e2e
script that shows it (roadmap M9.1). Each message names the file and line,
the step, or the producer involved, and says what to do next where that
isn't plain from the problem itself. `TestFailureChecklist` checks that
every script named here exists and contains its excerpt.

genx exits 1 on a failure and 2 on a usage error. All problems in one
manifest or metafile are reported at once. A run stops at the first step
that fails. Producers write their own files as they run; genx writes the
generators' files and `genx.lock` only after every step succeeded and every
output passed the checks below.

Scripts are in [internal/e2e/testdata/script](../internal/e2e/testdata/script).

## Usage

| Failure | What genx says | Script |
|---------|----------------|--------|
| Unknown command | `unknown command "frobnicate"` | [help](../internal/e2e/testdata/script/help.txtar) |
| Arguments to a command that takes none | `genx check: takes no arguments` | [help](../internal/e2e/testdata/script/help.txtar) |
| No go.mod above the working directory | `no go.mod in` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| No manifest at the module root | `no genx.yaml in the module root` | [no_manifest](../internal/e2e/testdata/script/no_manifest.txtar) |
| Interrupted (Ctrl-C): the running producer is stopped; a second Ctrl-C ends genx at once | `genx: interrupted; files already written stay, so run genx again` | [interrupt](../internal/e2e/testdata/script/interrupt.txtar) |

## The manifest and metafiles

| Failure | What genx says | Script |
|---------|----------------|--------|
| A manifest newer than genx | `genx.yaml:1: version: version 2 is newer than this genx supports; upgrade genx` | [versions](../internal/e2e/testdata/script/versions.txtar) |
| A metafile newer than genx | `genx.meta.yaml:1: version: version 2 is newer than this genx supports` | [versions](../internal/e2e/testdata/script/versions.txtar) |
| A duplicate YAML key (stops reading at once) | `genx.yaml:6: producers[0].emits: duplicate key "emits"` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| An unknown field | `producers[0].colour: unknown field` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| An invalid producer or generator name | `is not a valid producer name (a lowercase letter, then lowercase letters, digits and -)` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| An empty command | `producers[0].run: must not be empty` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| A timeout that isn't a duration | `producers[0].timeout: invalid duration "soon"` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| One path in emits twice, in one producer or two | `duplicate emit path "zz_a.json" (also line 5)` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| One generator switched on twice | `duplicate generator "x" for package "example.com/app/bad"` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| A domain that isn't `name/vN` | `invalid domain (want name/vN, such as labels/v1)` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| A generator with both or neither of `template` and `emits` | `generators.bad: exactly one of template and emits` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| A `doc` longer than one line | `generators.bad.doc: must be one line` | [manifest](../internal/e2e/testdata/script/manifest.txtar) |
| A refused jq function, when the metafile is read | `"now" is not allowed in a query (it reads the clock)` | [failures](../internal/e2e/testdata/script/failures.txtar) |

The other shape rules (types, empty values, template names) give messages of
the same form; `internal/manifest` and `internal/meta` test each one, and
`internal/yamlx` tests what applies to both files: a broken second document,
and aliases that expand to too many values.

## Parameters (amendment A4)

| Failure | What genx says | Script |
|---------|----------------|--------|
| A metafile's parameter name isn't `[a-z][a-z0-9_]*` | `params.Dir: "Dir" is not a valid parameter name` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| A parameter without `doc` | `params.nodoc.doc: required` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| A parameter without `default` (null must be written) | `params.nodefault.default: required` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| A parameter declared as a value, not `{doc, default}` | `params.flat: must be a mapping` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| A `use` entry sets a parameter the package doesn't declare (the message lists those it does) | `example.com/app/gen declares no parameter "colour"` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| …in a package that declares none | `example.com/app/plain declares no parameter "dir"` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| A value whose JSON type differs from a non-null default's | `params.dir: must be a string, as its default "models" is` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| …null for a parameter with a non-null default | `must be a string, as its default "" is` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| The map form of `use` without `generators` | `example.com/app/gen.generators: required` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| An unknown field in the map form | `example.com/app/gen.param: unknown field` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| `params` that isn't a mapping | `params: must be a mapping of parameter names to values` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |
| A `use` entry that is neither a list nor a mapping | `must be a sequence of generator names, or a mapping` | [params_failures](../internal/e2e/testdata/script/params_failures.txtar) |

All of an entry's parameter problems are reported at once, with the line
in `genx.yaml`. `genxtest.RenderParams` gives the same messages, located
as `params.<name>` and naming the metafile instead of the package.

## Finding generators

| Failure | What genx says | Script |
|---------|----------------|--------|
| A `use` package the go command can't find | `the go command can't find this package:`, then the build-tagged blank import to add | [use_missing](../internal/e2e/testdata/script/use_missing.txtar) |
| A package `go mod tidy` dropped | `no required module provides package example.com/labels` | [modules_untidy](../internal/e2e/testdata/script/modules_untidy.txtar) |
| A package without a metafile | `the package has no genx.meta.yaml, so it ships no generators; name the package that does` | [use_missing](../internal/e2e/testdata/script/use_missing.txtar) |
| A command's directory (`package main`) | `a command (package main) can't ship generators; move its genx.meta.yaml to an importable package` | [use_missing](../internal/e2e/testdata/script/use_missing.txtar) |
| A generator the metafile doesn't have | `no generator "labels" in its genx.meta.yaml (it has: read, write)` | [use_missing](../internal/e2e/testdata/script/use_missing.txtar) |

## Ordering the steps

| Failure | What genx says | Script |
|---------|----------------|--------|
| Steps that need each other's domains | `these steps need each other's domains: producer a (needs b/v1, written by producer b)` | [plan](../internal/e2e/testdata/script/plan.txtar) |
| A domain nothing writes | `needs labels/v1, which no step writes; add what writes it to genx.yaml, or remove what needs it` | [plan](../internal/e2e/testdata/script/plan.txtar) |
| First run only: a producer with `needs` writes a domain a step already read | `run genx again: the second run reads the domain from the file` | [plan](../internal/e2e/testdata/script/plan.txtar) |
| A domain in another major version than a step needs | `says labels/v2; upgrade the module that writes it, or the one that reads it` | [producers](../internal/e2e/testdata/script/producers.txtar) |

## Producers

| Failure | What genx says | Script |
|---------|----------------|--------|
| The command fails (its output is shown first, prefixed with its name) | `producer broken: fail some args: exit status 3` | [producers](../internal/e2e/testdata/script/producers.txtar) |
| The command isn't installed | `"genx-no-such-command" isn't installed, or isn't on PATH; install it, or run a module's tool with [go, tool, ...]` | [producers](../internal/e2e/testdata/script/producers.txtar) |
| It runs past its timeout | `timed out after 200ms (raise its timeout in genx.yaml)` | [producers](../internal/e2e/testdata/script/producers.txtar) |
| It didn't write a file it declares | `declared in emits, but the producer didn't write it` | [producers](../internal/e2e/testdata/script/producers.txtar) |
| A declared file isn't a JSON object | `not an object, so it can't name its domain (a foreign file needs "domain" in the manifest)` | [producers](../internal/e2e/testdata/script/producers.txtar) |
| A declared file has no `domain` | `no "domain" field (a foreign file needs "domain" in the manifest)` | [producers](../internal/e2e/testdata/script/producers.txtar) |
| A labels file breaks a `labels/v1` rule | `labels[0].var: "func" is a Go keyword` | [labels_check](../internal/e2e/testdata/script/labels_check.txtar) |

The conformance files in [testdata/labels](../testdata/labels) hold one
invalid file per `labels/v1` rule, each with its message.

## Generators

| Failure | What genx says | Script |
|---------|----------------|--------|
| A template the metafile names isn't there | `template: gone.tmpl isn't in the package's directory, next to genx.meta.yaml` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| A template is a symlink that leads out of the package's directory | `template: openat link.tmpl: path escapes from parent` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| A query calls `error` (here, i18n-style: one key, two texts) | `units: one key, two texts: title`, then `(input: a/zz_a.labels.json, b/zz_b.labels.json)` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| The template reads a key the unit lacks | `template: missing.tmpl:5:18: executing "missing.tmpl" at <.kind>: map has no entry for key "kind"` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| The template makes Go that doesn't parse (the lines around the error follow) | `zz_out.go: 11:1: expected operand, found '}'` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| A Go output without the generated-code header | `a generated Go file must start with "// Code generated ... DO NOT EDIT."` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| A unit of an `emits` generator isn't an object | `(each unit must be an object with "domain": "labels/v1")` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| A unit names another domain | `says domain other/v1; the generator emits labels/v1` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| An emitted file breaks its domain's rules | `zz_x.labels.json: package: required` | [failures](../internal/e2e/testdata/script/failures.txtar) |
| `out` gives more than one value | `out gave 2 values for a unit; it must give one path` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| `out` gives something other than a string | `out gave a number; it must give a path as a string` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| A generator's queries run past 30 s | not scripted: it takes 30 s; `internal/query` tests the timeout with a short one | — |

## Where outputs may go

| Failure | What genx says | Script |
|---------|----------------|--------|
| A path that leaves the module | `must not contain ".."` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| An absolute path | `must not be absolute` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| A base name without `zz_` | `the base name must start with zz_` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| A dot-directory, such as `.github` | `element ".github" must not start with '.'` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| A character outside `A–Z a–z 0–9 . _ -` | `contains invalid character ' '` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| A Windows reserved name | `"aux" is a reserved name on Windows` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| Two steps write one file, compared without case | `writes zz_a.go, which example.com/app/gen:upper writes too` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| A generator's file is one a producer declares, compared without case | `writes zz_Config.json, which producer config writes too` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |
| A symlink that leads outside the module | `writing link/zz_x.go:` | [out_refusals](../internal/e2e/testdata/script/out_refusals.txtar) |

## Files genx didn't write

| Failure | What genx says | Script |
|---------|----------------|--------|
| An output path holds a file genx didn't write | `exists and isn't in genx.lock, so genx didn't write it; move it away, then run genx again` | [leftovers](../internal/e2e/testdata/script/leftovers.txtar) |
| A Go output lost its header, so it may hold edits | `has lost its generated-code header, so it may hold edits; restore the header or delete the file` | [leftovers](../internal/e2e/testdata/script/leftovers.txtar) |
| A leftover lost its header | `is no longer generated, but it has lost its generated-code header` | [leftovers](../internal/e2e/testdata/script/leftovers.txtar) |
| `genx.lock` names a file twice | `a/zz_A_consts.go is listed twice` | [lock](../internal/e2e/testdata/script/lock.txtar) |
| A `genx.lock` line has the wrong shape | `want "<path> <package>:<generator>"` | [lock](../internal/e2e/testdata/script/lock.txtar) |
| A `genx.lock` path isn't clean | `isn't clean (use "a/zz_a_consts.go")` | [lock](../internal/e2e/testdata/script/lock.txtar) |
| …each of the three above, then | `genx.lock is generated: restore it from version control, or delete it and run genx` | [lock](../internal/e2e/testdata/script/lock.txtar) |

## genx check

`genx check` lists every problem it finds, then `run genx`.

| Failure | What genx says | Script |
|---------|----------------|--------|
| An output differs from what genx would write | `stale     zz_one.go (example.com/app/gen:one)` | [check](../internal/e2e/testdata/script/check.txtar) |
| An output is missing | `missing   zz_one.go (example.com/app/gen:one)` | [check](../internal/e2e/testdata/script/check.txtar) |
| A file nothing produces any more | `leftover  zz_old.go (example.com/app/gen:old wrote it; nothing does now)` | [check](../internal/e2e/testdata/script/check.txtar) |
| `genx.lock` is out of date | `stale     genx.lock` | [check](../internal/e2e/testdata/script/check.txtar) |
| A producer's own check fails | `cfg: failing: cfg.json is stale` | [check](../internal/e2e/testdata/script/check.txtar) |
| A producer without a check (a note, not a failure) | `cfg: not checked (no check command)` | [check](../internal/e2e/testdata/script/check.txtar) |

## Not scripted

These come from the operating system or the go command, and genx passes them
on with the step and file they concern:

- reading or writing a file (permissions, a full disk): `<step>: writing
  <path>: <error>`, `deleting <path>, which <step> wrote: <error>`;
- `go list` failing for a reason other than a missing package, shown with
  the go command's own output.

genx writes each file to a temporary `.genx-*` file next to it, then renames
it, so a crash never leaves half a file; at worst a temporary file stays
behind. A run that stopped before writing `genx.lock` recovers: a file that
already holds exactly what genx would write counts as genx's.

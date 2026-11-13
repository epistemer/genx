# ADR 0005: The labels pilot by hand (spike S3)

| | |
|---|---|
| Status | Accepted; the two format changes are proposed for the freeze (M1.6) |
| Date | 2026-11-13 |
| Roadmap item | M1.3 |
| Deciders | Claude, as owner, who ran the spike |

## Context

ADR 0001 drafts the manifest and the metafile. S3 runs them for real once:
cfgx's zp example, through i18n's generator, with a throwaway driver, and
checks that the result is the file cfgx writes today.

## What was tried

- **The producer.** cfgx M11.1 doesn't exist yet, so a 70-line program
  stood in for it: it reads the `zz_settings_i18n.go` that cfgx's
  `App.I18n()` wrote for `examples/zp` (cfgx commit `5a1759c`) and writes
  `internal/settings/zz_settings.labels.json` in `labels/v1`. 62 labels, 48
  with a variable, in the file's order. The manifest runs it as a producer,
  so the driver really executes a process and reads what it declared.
- **The generator.** A copy of i18n with `genx.meta.yaml` and
  `labels.go.tmpl` in its root package directory (ADR 0003). The query and
  template are i18n ADR 0003's, unchanged except for the input: the query
  starts from `.["labels/v1"]`, since review 0002 made the input an object
  keyed by domain.
- **The driver.** 249 lines: strict YAML for both files, producers run with
  `exec` (no shell), each declared file decoded with `UseNumber` and checked
  for its domain, `go list -f '{{.Dir}}'` for the `use` entry, gojq under a
  timeout, `text/template` with `missingkey=error`, `go/format`, and a write
  only when the bytes change.

## Results

- The generated `zz_settings_i18n.go` has **the same 117 lines of
  declarations** as cfgx's file. Only the first comment differs (the
  generator's header instead of cfgx's), and cfgx's explanatory comment is
  gone.
- The app builds against i18n v0.1.0's code.
- A run takes 0.27 s, most of it `go list`. The second run writes nothing.
- **`go mod tidy` removed i18n from the app's `go.mod`** before the first
  run, because nothing imported it yet. ADR 0003 records the fix.

## What the drafts needed

Every field the pilot used is in ADR 0001 §5 or the roadmap's drafts,
except two the drafts already show but §5 doesn't name: `version` and
`doc`. Two findings change the drafts:

1. **`vars` repeats the units query.** i18n's units query already builds
   the object the template reads: `vars` only copied four of its fields
   (`package: .package`, `vars: .vars`, ...). Anything a var computes, the
   units query can compute, and ADR 0001's reason for vars (a value of any
   shape, one per name) holds for a unit too. **Proposed for the freeze:**
   drop `vars`; each unit is the template's data, and `out` is a query over
   the unit.
2. **`use` names a package**, not a module (ADR 0003).

Smaller observations, for M1.6 to settle:

- The header line is written by the template (`// Code generated from
  labels by github.com/epistemer/i18n. DO NOT EDIT.`). genx can check that a
  `.go` output starts with a line matching Go's convention,
  `^// Code generated .* DO NOT EDIT\.$`, instead of writing its own.
- In the metafile, `units` is a multi-line YAML block holding the whole
  query with its `def`s. That reads well; `defs` (M3.5) only matters for
  sharing between generators.
- The driver needed nothing from cfgx or i18n beyond their files.

## Decision

The drafts work for the first real case. ADR 0001 §5 holds, with the two
changes above proposed for the freeze. The throwaway driver is not kept: M2
builds the real one package by package.

## Consequences

- The pilot's query and template become i18n's generator once the format is
  frozen (i18n M6.1). They need only the `.["labels/v1"]` start.
- M8.3's test ("the same declarations as before") is known to pass in
  principle.

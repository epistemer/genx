# ADR 0003: Finding metafiles (spike S1)

| | |
|---|---|
| Status | Accepted; the placement rule is proposed for the freeze (M1.6) |
| Date | 2026-11-13 |
| Roadmap item | M1.1 |
| Deciders | Claude, as owner. Gemini 3.8 Flash (High) ran the experiments (about 410k tokens); Claude reproduced the two that decide the rule |

## Context

An app's manifest switches on generators that other modules ship. The driver
runs inside the app and must find each module's metafile and templates in
every way an app gets its dependencies, without running code from them and
without the network when the app works offline.

## What was tried

A library `example.com/lib` served from a local file proxy, with two
placements: **A**, `genx.meta.yaml` and a template in the module's root
package directory, next to `lib.go`; **B**, a folder `genx/` with no Go
files. An app requiring it, in eight modes.

| Mode | A | B |
|------|---|---|
| Module cache | found | found |
| `replace ../lib` | found | found |
| `go.work` | found | found |
| `go mod vendor`, `-mod=vendor`, `GOPROXY=off` | found | **not found** |
| From `go generate` | found | found (not vendored) |
| Required only as a `tool` whose command is a subpackage, vendored | **not found** | **not found** |
| Required, no package imported yet, module cache | found | found |
| Required, no package imported yet, vendored | **not found** | **not found** |
| Warm cache, `GOPROXY=off` | found | found |

What decides it:

- `go mod vendor` copies every file of a package directory that holds Go
  files, including YAML and templates, and skips a directory with no Go
  files unless an `//go:embed` names it. It vendors only packages the build
  imports. (Claude reproduced this.)
- Under vendoring `go list -m` gives no `Dir`, and `go list -m all` fails.
  `go list -f '{{.Dir}}' <package>` resolves to `vendor/<package>` in every
  mode.
- **`go mod tidy` drops a requirement that nothing imports.** The labels
  pilot (S3, ADR 0005) hit this on its first run: before genx writes the Go
  that imports i18n, tidy removes i18n from `go.mod`, and `go list` fails.
- Timing: one `go list` call costs about 170 ms on this machine, cold or
  warm (most of it is starting the go command; reading `vendor/modules.txt`
  directly takes 0.02 ms). One call can list every `use` entry at once.

## Options

| # | Option | For | Against |
|---|--------|-----|---------|
| 1 | Placement B, a data folder | Keeps YAML out of Go package directories | Lost under vendoring |
| 2 | Placement A, files in a package directory, found by `go list` on the **package** | Works in all modes the app builds in; one call for all entries | The package must be imported (or a tool) for vendoring |
| 3 | `genx vendor` copies switched-on metafiles into the app | Works offline with anything | A second copy to keep current; the kill criterion's fallback |

## Decision

Option 2. A metafile and its templates live in the directory of a Go
package, and the manifest's `use` names that **package's import path**. genx
resolves every entry with one `go list -f` call on the packages, run in the
module root, which takes the app's own `-mod`, `go.work` and `replace`
settings from the go command.

When a package can't be listed, the message says why it usually happens:
the app doesn't import it yet, so `go mod tidy` removed the requirement, or
`go mod vendor` didn't copy it. The fix is a blank import in a file the app
keeps, which is the same thing a `tools.go` file did before `tool`
directives:

```go
//go:build genx

package app

import _ "github.com/epistemer/i18n"
```

The build tag keeps the import out of the binary; `go mod tidy` and
`go mod vendor` consider all build tags, so the requirement and the vendored
copy stay.

The kill criterion isn't met: placement A works under vendoring offline. No
`genx vendor` command.

## Consequences

- `use` names packages, not modules. That changes the manifest draft and
  goes to the freeze. For i18n the package is its root package, which every
  app that renders text already imports.
- A package that ships a generator puts `genx.meta.yaml` and its templates
  next to its Go files. i18n ADR 0003's open question about where the files
  sit is answered.
- `go list` runs once per genx run, about 170 ms. M9.2 measures it in a
  real app.
- genx works on one module, the one whose `go.mod` is nearest above the
  working directory. Under `go.work`, each module that generates has its own
  manifest.
- Roadmap M6.1 and M6.2 become one item's worth of work: the same `go list`
  call in every mode, with e2e scripts for module cache, `replace`,
  `go.work`, vendoring offline, and the missing-import message.

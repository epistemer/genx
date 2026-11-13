# ADR 0002: The test harness and its dependency

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-13 |
| Roadmap item | M0.3 |
| Deciders | Claude, as owner |

## Context

The definition of done asks for a `testscript` case for every behaviour a user
can see, with the exact text of each error, and golden files for messages.
cfgx already has both helpers, in `internal/e2e` and `internal/golden`, which
another module can't import.

## Options

| # | Option | For | Against |
|---|--------|-----|---------|
| 1 | `github.com/rogpeppe/go-internal` (`testscript`, `diff`), as cfgx and i18n use | Known in neozp; scripts read like shell sessions; the same version as cfgx | A test-only dependency |
| 2 | Plain Go tests that call `cli.Main` | No dependency | Scripts with files and expected output become long Go literals |

## Decision

Option 1, at the version cfgx pins (v1.16.0). The CLI lives in
`internal/cli` as `Main(args, stdout, stderr) int`, so `cmd/genx` and the
scripts run the same code: `testscript.Main` registers `genx` as a command.
Each script runs in a fresh module, `example.com/app`, unless it writes its
own `go.mod`. The golden helper is copied from cfgx unchanged.

The CLI takes no command library, not cobra: genx has three commands and no
settings of its own. When a command needs flags, it uses the standard `flag`
package.

## Consequences

- Only test code imports go-internal; `cmd/genx` doesn't link it.
- Scripts that need the go command (M6) get the host's caches through the
  environment the harness copies, as in cfgx.
- M6 serves test modules from `internal/e2e/testdata/mod` with go-internal's
  `goproxytest`, so scripts exercise the module cache, `replace`, `go.work`
  and vendoring with no network. It brings `golang.org/x/mod` in as an
  indirect, test-only dependency.

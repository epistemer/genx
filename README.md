# genx

The code generation driver for neozp. Packages publish data about what they
generate (domain files), other packages ship generators that are only data
(queries and templates), and genx runs both in order, checks that generated
files are current, and removes what nothing produces any more.

```bash
go get -tool github.com/epistemer/genx/cmd/genx
go tool genx          # run every step; write what changed; delete leftovers
go tool genx check    # in CI: write nothing, fail if anything is stale
go tool genx list     # the steps in order, and generators not switched on
```

## Quickstart: an app on cfgx and i18n

This adds genx to an app whose settings cfgx generates, such as one made
with `cfgx init`, so that i18n turns every setting's description into a
translatable label. You need cfgx v0.2.0 or later and i18n v0.2.0 or later.

**1. Add the tools and the library.** In the app's module root:

```bash
go get -tool github.com/epistemer/genx/cmd/genx@latest
go get github.com/epistemer/cfgx@latest github.com/epistemer/i18n@latest
```

**2. Have cfgx write the labels.** In `internal/settings/schema`, add
`.Labels()` to the app:

```go
var App = dsl.App("shop", "shop", Log, Server).
	ConfigFile("shop", ".").
	Global(Log).
	Labels()
```

cfgx now also writes `zz_settings.labels.json`: the label of every group,
setting and enum value, as `labels/v1` data. It doesn't import i18n.

**3. Write the manifest.** Create `genx.yaml` next to `go.mod`:

```yaml
version: 1

producers:
  - name: settings
    dir: internal/settings
    run:   [go, tool, cfgx, gen, ./schema]
    check: [go, tool, cfgx, gen, -check, ./schema]
    emits:
      - internal/settings/zz_settings.labels.json

use:
  - github.com/epistemer/i18n: [labels]
```

The producer is the command that ran from `go generate` before, and
`emits` names the labels file it writes. `use` switches on i18n's `labels`
generator, which turns every labels file into Go.

**4. Let genx run cfgx.** Delete the `//go:generate go tool cfgx gen
./schema` line in `internal/settings/settings.go`, and put this one in
`main.go`, under `package main`:

```go
//go:generate go tool genx
```

**5. Generate.**

```bash
go generate ./...
go mod tidy
```

genx runs cfgx, then writes `internal/settings/zz_settings_i18n.go` and
`genx.lock`, the list of files genx wrote. Commit `genx.yaml`,
`genx.lock` and every `zz_` file. `go mod tidy` keeps i18n, which the new
file imports. From now on, `go generate ./...` after a schema change runs
both steps.

**6. Use the labels.** Each setting and group now has an i18n key,
`settings.<path>`, with its description as the English text:

```go
settings.L10nServerAddr.T(ctx)  // "Address to listen on", or its translation
settings.L10n("server.addr")    // the same label, by the path the store's Keys gives
```

Translations are edited in the running app (see i18n's README).

**7. Keep it current.** In CI, run `go tool genx check`: it writes nothing
and fails, naming each file, when a generated file is stale, missing or
left over. Or check from a test:

```go
func TestGenerated(t *testing.T) { genxtest.AssertGenerated(t) }
```

What's next:

- Other packages' generators are switched on the same way, in `use`.
  `go tool genx list` shows the ones in your build that aren't. A
  generator that takes parameters, such as the directory it writes to, gets
  them from the entry's map form:
  `- <package>: {generators: [...], params: {dir: internal/db}}`.
- Writing a producer: [docs/guides/producer.md](docs/guides/producer.md).
- Writing a generator: [docs/guides/generator.md](docs/guides/generator.md),
  with a complete, tested example in [examples/labels](examples/labels).
- When something fails: [docs/failures.md](docs/failures.md) lists every
  message genx gives.

## Running it from go generate

Put the directive in a file of the module's root package, or of any package
genx doesn't write into:

```go
//go:generate go tool genx
```

`go generate` lists a package's files before it runs, so genx deleting a
leftover file in the directive's own package would make it fail.

## Timings

Measured on 2026-11-13 with genx `v0.1.0-rc.3`, Go 1.27.4 (installed as a
snap) on Linux, a 4-thread Intel i5-5300U from 2015. Each figure is the
median of five runs, in wall time.

**cfgx's `examples/zp`**: 62 labels for its groups, settings and enum values, one producer (cfgx)
and one generator (i18n's `labels`).

| Run | Time |
|-----|------|
| `go tool genx`, nothing changed | 2.8 s |
| `go tool genx check` | 2.8 s |
| `go tool genx` with no generated files yet | 2.0 s |
| `cfgx gen ./schema` on its own, for comparison | 2.5 s |
| Cold: `go tool genx` with an empty build cache (three runs) | 45–47 s |

The producer takes the time: cfgx loads the schema package with the go
command. genx's own share is about 0.3 s, most of it one `go list`. A cold
run compiles genx, cfgx and their dependencies once.

**zp's enums from its sqlc request**: the generator from spike S4 (ADR
0006), as a trivial generator over zp's 5.4 MB `codegen_request.json`,
which the manifest names as a foreign file. Its output is 931 lines,
identical to the hand-written generator's.

| Run | Time |
|-----|------|
| `genx`, nothing changed | 0.36 s |
| `genx check` | 0.35 s |
| `genx` with no generated file yet | 0.36 s |
| The same with a second generator that emits a labels file (112 labels) | 0.36 s |
| `go list` on its own, for comparison | 0.17 s |

Decoding the 5.4 MB request, the query, the template and gofmt take about
0.2 s; the rest is starting the go command. Neither case asks for parallel
steps or caching yet (roadmap backlog, item 8).

## Documents

The contract is [docs/interface.md](docs/interface.md). The design and the
plan:

- [docs/adr/0001-generation-pipeline.md](docs/adr/0001-generation-pipeline.md):
  why a driver, and the convention it runs.
- [docs/roadmap.md](docs/roadmap.md): the order of work to `v0.1.0`.
- [docs/status.md](docs/status.md): the log of working sessions.
- [CHANGELOG.md](CHANGELOG.md): what each release changed.

Part of neozp, alongside [cfgx](https://github.com/epistemer/cfgx) and
[i18n](https://github.com/epistemer/i18n).

# genx

The code generation driver for neozp. Packages publish data about what they
generate (domain files), other packages ship generators that are only data
(queries and templates), and genx runs both in order, checks that generated
files are current, and removes what nothing produces any more.

## Running it

```bash
go get -tool github.com/epistemer/genx/cmd/genx
go tool genx          # run every step; write what changed; delete leftovers
go tool genx check    # in CI: write nothing, fail if anything is stale
go tool genx list     # the steps in order, and generators not switched on
```

To run it from `go generate`, put the directive in a file of the module's
root package, or of any package genx doesn't write into:

```go
//go:generate go tool genx
```

`go generate` lists a package's files before it runs, so genx deleting a
leftover file in the directive's own package would make it fail.

The contract is [docs/interface.md](docs/interface.md). The design and the
plan:

- [docs/adr/0001-generation-pipeline.md](docs/adr/0001-generation-pipeline.md):
  why a driver, and the convention it runs.
- [docs/roadmap.md](docs/roadmap.md): the order of work to `v0.1.0`.
- [docs/status.md](docs/status.md): the log of working sessions.

Part of neozp, alongside [cfgx](https://github.com/epistemer/cfgx) and
[i18n](https://github.com/epistemer/i18n).

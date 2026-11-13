# ADR 0007: `check` for producers (spike S5)

| | |
|---|---|
| Status | Accepted; the behaviour is proposed for the freeze (M1.6) |
| Date | 2026-11-13 |
| Roadmap item | M1.5 |
| Deciders | Claude, as owner. Gemini 3.8 Flash (High) ran the measurements (about 395k tokens) |

## Context

`genx check` writes nothing and fails on stale output. Generators are easy:
genx renders in memory and compares. A producer is someone else's program.
Review 0002 ruled out running it and putting the files back.

## What was tried

| Producer | Up to date | Hand-edited output | Input changed | Output deleted | Wrote anything? |
|----------|-----------|--------------------|---------------|----------------|-----------------|
| `cfgx gen -check` | exit 0, silent | exit 1, names the file | exit 1, lists the files | exit 1, names it | No (tree hash equal) |
| `sqlc diff` | exit 0, silent | exit 1, a diff | exit 1, a diff; covers process plugins' files | exit 1, **silent** | No |

For a producer with no check mode:

| Strategy | Cost | Problems |
|----------|------|----------|
| Run it in a temporary copy of the module | 2 ms for a small module; 530 ms to copy zp (322 MB), 220 ms filtered | `replace ../x` breaks, `go.work` is lost, absolute paths differ; filtering large files dropped binaries zp's sqlc plugins need |
| Redirect its output | – | Neither cfgx nor sqlc can write Go elsewhere |
| Trust it: check that each declared file exists, parses and names its domain; check every generator downstream | none | Can't see output that is stale because its input changed and nobody re-ran it |

## Decision

- A producer with a `check` command: `genx check` runs it in place, in the
  producer's `dir`, and a non-zero exit fails the check with the producer's
  output. Its own message is its own (`sqlc diff` says nothing for a deleted
  file); genx adds a line naming the producer and the command.
- A producer without one is **trusted**: its declared files must exist,
  parse and name their domain, and everything downstream is checked
  against them. `genx check` prints one line per trusted producer, so the
  gap is visible.
- No temporary copies. No manifest option to choose a strategy in `v0.1.0`;
  the friction log (M8.4) can ask for one.

## Consequences

- M5.3 has two cases instead of three.
- A producer that wants to be checked ships a check mode. cfgx has one;
  the guide for producer authors (M7.4) asks for it.
- The manifest's `check` field stays optional.

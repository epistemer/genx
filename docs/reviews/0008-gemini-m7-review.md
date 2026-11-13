# Review 0008: M7, labels/v1, genxtest and the guides

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewers | Claude reviewed `internal/labels` and the conformance files Gemini 3.8 Flash (High) wrote (about 455k tokens); Flash (High) reviewed M7 as a whole (about 98k) |

## Claude on Gemini's labels checks

One `record` helper checks every `labels/v1` document where it enters: a
producer's file under `genx` and `genx check`, and a generator's emitted
document. 38 rules, each with an invalid conformance file and its exact
message. Nothing changed.

## Gemini on M7

| # | Finding | Verdict |
|---|---------|---------|
| 1 | `package.dir: "."` rejected | Rejected: `paths.Dir` allows `.`; `testdata/labels/valid/root_dir.json` passes |
| 2 | The example's template declared `Texts`, which §11 doesn't let a renderer declare | Accepted: it declares `l10n` and `L10n` now |
| 3 | `AssertGenerated` needs the module root as working directory | Rejected: genx walks up to the nearest `go.mod` |
| 4 | A root package makes `./zz_x.go` | Rejected: `out` is cleaned before it's checked (§7) |
| 5 | One key in two files breaks the example's map literal | Accepted: its query keeps one label per key and refuses one key with two texts |
| 6 | `Render` can't test a generator's failures | Open: part of the `genxtest` API question for epistemer |
| 7 | `AssertGenerated` logged the producers' output apart from the failure | Accepted: it's in the failure message |
| 8 | The example producer failed without its directory | Accepted |

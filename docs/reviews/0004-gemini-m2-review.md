# Review 0004: M2, the walking skeleton

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewers | Claude reviewed the leaf packages Gemini wrote (M2.1–M2.4); Gemini 3.8 Flash (High) reviewed the runner Claude wrote (M2.5), about 250k tokens |
| Scope | `internal/{yamlx,paths,domain,manifest,meta,query,render}` (Gemini, Flash High, about 1.1M tokens to write); `internal/run`, the CLI and the e2e scripts (Claude) |

## Claude on Gemini's packages

- **query:** the refusal walk named each part of gojq's syntax tree by hand.
  It caught every case tried, but a field it didn't name could hide a call,
  so it now walks every exported field by reflection, and also refuses
  `{$ENV}` (object shorthand) and refused names as `def` parameters. Twenty
  more refusal cases are tested.
- **yamlx:** one staticcheck finding (S1017), fixed.
- **paths, manifest, meta, domain, render:** read against interface.md §3,
  §4, §6, §7 and §10; no changes.

## Gemini on the runner

| # | Finding | Verdict |
|---|---------|---------|
| 1 | Nil `Stdout`/`Stderr` panic | Accepted: default to `io.Discard` |
| 2 | Timeouts misattributed when the run itself is cancelled; a query timeout reported as a query error | Accepted |
| 3 | Missing packages reported in map order | Accepted: manifest order |
| 4 | Multi-line `go list` errors cut off | Accepted: `go list -json` |
| 5 | A package that exists but fails to load got the blank-import advice | Accepted: the two cases are told apart |
| 6 | Producers ran before duplicate `emits` were found | Accepted: every declared path is claimed before anything runs |
| 7 | A failed removal of the temporary file was silent | Accepted |
| 8 | Slash and OS paths mixed in `os.Root` calls | Rejected: OS paths go to `os.Root`, slash paths to messages |
| 9 | `prefixWriter` returned `len(b)` with an error | Accepted |
| 10 | An emitted document was recorded before its path was claimed | Accepted |
| 11 | A directory named `go.mod` counted as a module root | Accepted |

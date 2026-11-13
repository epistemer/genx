# Review 0009: M9.3, genxtest and both file formats against the interface

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewer | Gemini 4 Hydrogen (High), about 464k tokens (Argon was out of quota). Claude checked each finding against the code before fixing it |
| Scope | The manifest (§2, §3, §7, §9), the metafile (§2, §4–§6), `genxtest` (A1) and `labels/v1` (§11), as of the merge of M9.1 |

## Findings

| # | Clause | Finding | Verdict |
|---|--------|---------|---------|
| 1 | §10, A1 | `Render` didn't compare an input's major version with the generator's `needs`: a `labels/v2` fixture fed `[]` to a `labels/v1` generator | Accepted: it fails the test, naming the file and both versions |
| 2 | §5 | The YAML decoder marked every alias it passed and never unmarked it, so an anchor reached on two paths was called circular | Accepted, confirmed with `a: &a 1, b: &b [*a], c: *b, d: *b`. Cycles are now nodes on the current path. The old bug was also the only thing stopping a file whose aliases expand exponentially, so both YAML readers now refuse expansion past a budget (A3) |
| 3 | §2 | A broken second document in a manifest or metafile was ignored | Accepted, confirmed: it's an error with its line |
| 4 | §2, §3 | `use` accepted a `package main` directory; package resolution didn't use `-tags=genx` as `genx list` does | Accepted, both |
| 5 | §2 | A non-Go output kept CRLF line endings that a unit's text brought in | Accepted: `render.Finish` turns them into LF |
| 6 | A1, §1.5 | `genxtest` had no tests in the main module (only `examples/labels`) | Accepted: `genxtest_test.go` covers input order, foreign inputs, `emits`, four failures, and that `AssertGenerated` writes nothing and fails on a stale file |
| 7 | §3 | No test ran a producer with a `dir` | Accepted: `producers.txtar` |
| 8 | §4, §6 | Empty renders and a `units` query with no output, deleting the earlier file, weren't run end to end | Accepted: `outputs.txtar` |
| 9 | §8 | A leftover already gone from the disk, and a rename that changes only case, weren't tested | Accepted: `outputs.txtar` |
| 10 | §9 | `check` rendering a generator over another's in-memory output wasn't tested | Accepted: `check_chain.txtar` |
| 11 | §7, §9 | A generator writing a producer's file wasn't tested | Accepted: `out_refusals.txtar` |
| 12 | §2, §4 | A missing template, and a template symlink leading out of the package, weren't tested | Accepted: `failures.txtar`; the missing template's message now says where genx looked |
| 13 | §11, §5, A2 | The schema wasn't checked against the conformance files; `.yml` wasn't tested; only `GOPACKAGE` of A2's four variables was | Accepted: `TestConformanceSchema` (every valid file passes the schema, whose keywords the test's validator all implements), a `.yml` input in `inputs.txtar`, and the stand-in producer refuses all four |
| 14 | – | failures.md's "manifest newer than genx" row pointed at a metafile case | Accepted: `versions.txtar` has a manifest case |

## Checked and found correct

Gemini's checklist, which Claude spot-checked:

- **§2:** the module root and its error; `os.Root` for the module and each
  metafile's directory; strict YAML with file and line; `version` above 1.
- **§3:** every manifest field, its default and its rule; resolution through
  the go command; the blank-import advice; a repeated generator.
- **§4:** every metafile field; `defs` before every query; `template` or
  `emits`, not both.
- **§5:** the input's shape and path order; `[]` for a domain with no file;
  integer precision; YAML keys, timestamps, `.nan` and `.inf`; the 29
  refused names; `error` with its inputs; the 30 s budget.
- **§6:** `text/template` with `missingkey=error`; CRLF in templates;
  `go/format` with the lines around an error; the header; empty renders.
- **§7, §8:** the path rules; case-insensitive ownership; atomic writes;
  every check before any write; the lock's format; leftovers; never-replace.
- **§9:** the order of steps; `check` reporting everything and writing
  nothing; `list`; exit codes.
- **§11:** the schema, `internal/labels` and the conformance files agree.
- **A1:** `genxtest`'s exported API is A1's, word for word.

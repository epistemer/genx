# Review 0003: the interface before the freeze (M1.6)

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewer | Gemini 4 Hydrogen (High), through agy; about 59k tokens |
| Scope | The first draft of `interface.md`, ADRs 0003–0007 and `schema/labels.v1.json`, from a scratch folder |
| Triage | Claude. Every accepted finding is in the revised draft |

| # | Finding | Verdict |
|---|---------|---------|
| 1 | The `zz_` rule blocked foreign files and contradicted §10 | Accepted: `zz_` applies to `out` only; a producer's paths keep their names |
| 2 | A root package (`package.dir: "."`) couldn't be generated into | Accepted: `.` is allowed; `out` is cleaned before it's checked |
| 3 | `template` could name `../../etc/passwd`; reads didn't go through `os.Root` | Accepted: a file name with no `/`, read through `os.Root` at the metafile's directory; module reads through `os.Root` too |
| 4 | `genx list`'s "module graph" fails under vendoring | Accepted: it lists the build's packages (`go list all`), warns on a bad metafile it isn't asked to run |
| 5 | Case folding undefined; a case-only rename could delete the new file | Accepted: Unicode simple folding; the lock is compared without case; leftovers are deleted before writes |
| 6 | YAML types undefined; `*big.Int` encodes as a string | Accepted: YAML's JSON equivalent spelt out; genx's encoder for domain files is specified |
| 7 | `emits` was a list in the example and one domain in the table | Accepted: a string |
| 8 | Tie-breaking in the order undefined; duplicate `use` entries | Accepted: the ready step first in the manifest goes first; duplicates are errors |
| 9 | 30 s per query could exhaust memory; templates have no timeout | Accepted in part: 30 s per generator, checked between units; the interface says a template can't be stopped |
| 10 | Producers with `check` weren't validated; foreign files have no `domain` | Accepted: every producer's files are checked after `run` and under `check`, foreign files excepted |
| 11 | The schema's identifier pattern was ASCII and allowed keywords; genx's own checks were incomplete; unknown fields undefined | Accepted: the schema checks shape; genx checks identifiers with Go's rules; §11 lists genx's checks and says cross-file checks are the consumer's; unknown fields are ignored within a major version |
| 12 | Spaces broke the lock's format; stale/missing/leftover were used loosely; `check` didn't cover the lock | Accepted: no spaces in paths; the three words are defined; `check` fails on a lock that would change; preflight before any write |
| 13 | Path rules missed `<>"|?*`, defined "clean" loosely | Accepted: elements use `A–Z a–z 0–9 . _ -` only; clean means equal to `path.Clean`; reserved names checked on the stem, in any case |
| 14 | A `package main` can't be blank-imported | Accepted: a metafile's package must be importable |
| 15 | "A major version no step expects" blocked unconsumed domains and couldn't name a module | Accepted: only a domain a step needs, in another major, stops the run, naming the file and the step |
| 16 | Zero units, `out` arity, `defs` joining | Accepted |
| 17 | `missingkey=error` trips on optional fields | Accepted as documentation: the unit must hold every key the template reads, `null` if need be |
| 18 | Header check before or after gofmt; CRLF on disk; temporary files | Accepted: after formatting; CR ignored; temporary files in the target directory |
| 19 | Metafile strictness, `dir`/`timeout`/argv rules, domain-name syntax, one value per file, `check` aggregation, empty texts | Accepted: all specified. Empty texts are allowed |

# Review 0002: ADR 0001 and the roadmap (M0.5)

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewer | Gemini 4 Hydrogen (High), through agy; about 39k tokens. Argon, the usual model for a design review, was out of quota |
| Scope | ADR 0001 and the roadmap, read from a scratch folder holding only those two files |
| Triage | Claude. Every accepted finding is a roadmap change; four also amend ADR 0001 (its changes 9–12) |

| # | Finding | Verdict |
|---|---------|---------|
| 1 | A dependency's generator could write through a symlink, or create `.git/hooks/*` or a workflow | Accepted: M3.2 writes through `os.Root` and refuses dot-directories and module files; ADR change 12. Whether every output must be named `zz_*` goes to the freeze (M1.6) |
| 2 | The plan can't order generator → generator without knowing what a generator emits | Accepted: generators declare `emits` (ADR change 10, M3.3, M4.1); every unit renders before anything is written (M3.2) |
| 3 | The input is an array for one domain and an object for several, so adding a domain breaks queries | Accepted: always an object keyed by domain (ADR change 9, M3.1, the draft) |
| 4 | `check` can't see a multi-hop pipeline if each generator reads the disk; S5's "put the files back" writes to the tree | Accepted: generators render over an in-memory overlay (M5.3); S5's options are a temporary copy or refusal, never writing to the tree. A producer with `needs` still reads the disk under `check`: if its input is stale, `check` already fails on that input |
| 5 | Number precision and object key order through gojq | Accepted into S2: `UseNumber`, key order over 100 runs, precision in the kill criterion |
| 6 | Windows paths, reserved names, case-insensitive collisions, CRLF templates | Accepted: M3.2's path rules and case-insensitive collisions, M2.4 reads templates as LF, a new risk row |
| 7 | Store hashes in the lock to detect hand edits of any format | Rejected: hashes put every regeneration into the lock and make merge conflicts. `check` in CI already fails on a hand-edited output, and git keeps what a deletion removes. How a non-Go output is recognized as generated goes to M1.6 |
| 8 | S1 misses `tool` modules, modules not imported yet, and offline lookups | Accepted into S1 |
| 9 | Templates can't write valid JSON; empty renders and unused imports are undefined | Accepted in part: domain files are encoded from a query's result (ADR change 11); an empty render's meaning goes to M1.6. Unused imports stay the template's job, and gofmt's message shows them |
| 10 | gojq's `$ENV`, `import`, `debug` and runaway queries | Accepted into S2 and M2.3: loaders unset, these refused, a timeout through `RunWithContext` |
| 11 | M7.3 needs `check` (M5); M3.2's "never replace" needs the lock; M3.3 needs ordering; the lock isn't in the freeze | Accepted: M7 needs M5; "never replace a file the driver didn't write" moves to M5.2; M3.3 runs in `use` order until M4.1, whose e2e is multi-hop; the lock's format joins M1.6 |
| 12 | M3.6 doesn't compare across OSes; spikes' kill criteria are weak; S4 should cover table names | Accepted in part: golden outputs are shared by the three OSes in CI; S2's kill criterion adds memory, precision and stopping. Rejected: widening S4 to tables, which belong to the models package after `v0.1.0` |
| 13 | `use` can't name the app's own metafile | Accepted as a clarification: the app's own module path works in `use` |
| 14 | `go generate` from a package other than the root | Accepted into M1.6: how genx finds the module root |
| 15 | Non-atomic writes; unknown fields in metafiles; `go.work` scope | Accepted: temporary file and rename (M2.5). Unknown fields are already errors (M2.1, M2.2), and a newer metafile version names the module to upgrade (M6.4). genx works on one module; S1 records what `go.work` changes |
| 16 | Hand-written domain files, and globs in `emits` | Rejected for `v0.1.0`: a committed file with no producer can be a foreign file named in the manifest (M4.3); globs make `check` depend on a run. Both can come back from the friction log |

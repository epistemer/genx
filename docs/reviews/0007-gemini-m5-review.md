# Review 0007: M5, check, the lock and cleanup

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewer | Gemini 3.8 Flash (High), about 189k tokens |
| Scope | The lock, leftovers, the preflight before writes, `genx check` (written by Claude) |

| # | Finding | Verdict |
|---|---------|---------|
| 1 | A run that stops after writing its files but before the lock wedges the next run ("exists and isn't in genx.lock") | Accepted: a file that already holds exactly what genx would write is taken as genx's. A script removes the lock and runs again |
| 2 | Read errors other than "not found" counted as "nothing there" | Accepted: they stop the run, naming the step and the file |
| 3 | `check` stopped at a producer's missing file, dropping what it had found | Accepted: missing or broken producer files are problems like the rest; a later failure is reported after them |
| 4 | On macOS and Windows a case-only rename passed `check` | Accepted: genx looks files up by their directory listing, so a file on disk under another case is stale, not current |
| 5 | The lock accepted unclean paths | Accepted |
| 6 | The lock accepted duplicate paths and steps without `package:generator` | Accepted |
| 7 | A leftover already gone from disk isn't reported by name | Rejected: the lock is reported stale, which is the whole problem; naming a file that isn't there would mislead |
| 8 | On Linux an output could sit beside a hand-written file that differs only in case | Accepted, with 4: the preflight finds case variants too |

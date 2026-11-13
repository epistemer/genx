# Review 0005: M3 and M4, generators and the pipeline

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewers | Gemini 3.8 Flash (High) wrote `internal/plan` (about 346k tokens), which Claude reviewed and changed; Flash (High) reviewed the M3/M4 runner and plan diff (about 108k tokens) |
| Scope | `internal/plan`, its use in `internal/run`, `genx list` |

## Claude on Gemini's plan

- A producer's writes are unknown on a first run, so a step with needs
  could run before a producer that needs nothing and writes what it needs.
  Such steps now wait for unknown producers that need nothing; only an
  unknown producer with needs of its own can still turn out late, which
  `Done` reports with "run genx again".

## Gemini on the integration

| # | Finding | Verdict |
|---|---------|---------|
| 1 | `genx list` failed on a clean checkout ("needs labels/v1, which no step writes") and could disagree with a run | Accepted: `Order` uses the same readiness rules and reports cycles, not missing domains; a script runs `genx list` before the first run |
| 2 | A step that needs a domain it also writes deadlocked on itself | Accepted: a step doesn't wait for itself; it reads what the other writers wrote |
| 3 | A need nothing writes could pass when an unknown producer let the step run | Accepted: once every step has run, every need must have been written |
| 4 | A stale file on disk could make a false major-version error before its producer ran | Accepted: a domain read from a file a previous run left is a guess (`Peeked`); it orders steps but no error rests on it. Ordering goes by domain name, so a stale major still orders correctly |
| 5 | A major mismatch reported while an unknown producer was pending | Accepted, with 4 |
| 6 | Several writers where one is an unknown producer with needs | Rejected: the documented first-run case; `Done` says to run again, and the second run knows the domain |
| 7 | `genx list` doesn't show generators that `use` doesn't name | Deferred: that's M6.3 |
| 8 | A cycle message listed steps merely waiting behind the cycle | Accepted: only steps on a cycle are named |

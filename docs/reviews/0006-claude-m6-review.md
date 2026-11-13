# Review 0006: M6, finding and switching on generators

| | |
|---|---|
| Date | 2026-11-13 |
| Author | Gemini 3.8 Flash (High), about 451k tokens |
| Reviewer | Claude |

Gemini wrote M6.1–M6.4: a `goproxytest` module proxy for the e2e scripts,
scripts for the module cache, `replace`, `go.work`, vendoring offline (an
empty module cache and `GOPROXY=off`) and an untidy `go.mod`; `genx list`'s
"Not switched on" section; `plan.MismatchError`, which `genx` turns into a
message that names the files.

- **Kept:** Gemini found that `go list all` skips files behind
  `//go:build genx`, the blank import ADR 0003 recommends, so `genx list`
  passes `-tags=genx`. The workspace script also showed that `-mod=mod`
  can't be set under `go.work`, so the harness leaves `GOFLAGS` alone.
- **Changed:** the mismatch message says "says" for one file; a metafile
  that doesn't parse is shown with the parser's own `file:line:` message.
- **Noted:** `golang.org/x/mod` enters `go.sum` as go-internal's test-only
  dependency (ADR 0002 updated).

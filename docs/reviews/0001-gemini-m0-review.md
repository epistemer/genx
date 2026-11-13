# Review 0001: M0, the repository foundation

| | |
|---|---|
| Date | 2026-11-13 |
| Reviewer | Gemini 3.8 Flash (Medium), through agy; about 61k tokens |
| Scope | The M0 diff (module, CLI stub, e2e harness, CI), without `go.sum`, the license and the copied golden helper |
| Triage | Claude |

| # | Finding | Verdict |
|---|---------|---------|
| 1 | `actions/checkout@v7` and `setup-go@v7` don't exist | Rejected: cfgx and i18n run green on them |
| 2 | `go.sum` is missing | Rejected: left out of the diff on purpose |
| 3 | `-race` with cgo fails on Windows | Rejected: i18n's identical matrix is green on Windows |
| 4 | Passing the host's `PATH` hides testscript's `genx` | Accepted, defensively: `PATH` is left to testscript |
| 5 | The golden helper is missing | Rejected: left out of the diff on purpose |
| 6 | `''` doesn't escape a quote in testscript | Rejected: it does, and the script passes |
| 7 | `testscript.Main` passes a flag through `os.Args` on Windows | Not acted on: go-internal handles Windows; CI shows it |
| 8 | `gofmt` fails on CRLF checkouts | Rejected: `.gitattributes` forces LF |
| 9 | staticcheck runs on one job | Rejected: as in cfgx; it's platform-independent here |
| 10 | `go 1.26.0` should be `go 1.26` | Rejected: current toolchains write the patch |
| 11 | No `go build` step, so `cmd/genx` isn't linked | Accepted |
| 12 | Host `TEMP`/`TMP` override the script's temp dir | Accepted |
| 13 | The script doesn't check exit code 2 | Accepted: `cli_test.go` checks every exit code |
| 14 | ADR 0002 names `flag`, the code doesn't use it | Accepted: the ADR now says no library until a flag is needed |
| 15 | `/genx` isn't ignored | Accepted |

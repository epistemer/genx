# Friction log (M8.4)

What went wrong, or was awkward, when cfgx and i18n first generated through
genx (roadmap M8.2 and M8.3). Each entry ends as an amendment, a fix, or a
backlog item.

| # | Where | What happened | Outcome |
|---|-------|---------------|---------|
| 1 | i18n `examples/basic` under `go generate` | The directive sits in `package main`, so `go generate` set `GOPACKAGE=main`. genx passed it to cfgx, which takes its package name from `GOPACKAGE`, and cfgx wrote `zz_main.go` into `internal/settings` | Fixed in `v0.1.0-rc.3`: producers run without `go generate`'s variables (interface amendment A2) |
| 2 | cfgx `examples/zp` | `cfgxtest.AssertGenerated` finds cfgx's arguments in the package's `go:generate` line. Under genx that line runs genx from `main`, so the test fails | Documented in cfgx interface §5.5: an app on genx uses `genxtest.AssertGenerated`, which runs `cfgx gen -check`. cfgx backlog: cfgxtest could read the producer's `check` from `genx.yaml` |
| 3 | i18n `labels/` tests | `genxtest.Render` fails the test on any error, and the generator's checks (one key, two texts) needed testing | No API change (A1 stands): wrap `t` in a type that records `Fatalf`. The generator guide shows how |
| 4 | Both manifests | A producer's `run` and `check` repeat the same argv with one flag added (`-check`), and the app spells out cfgx's command | Backlog item 2 (producers described by their modules) already covers it; nothing for `v0.1.0` |
| 5 | cfgx `examples/zp/bench` | A module that requires `examples/zp` needed `go mod tidy` once zp required i18n and genx; cfgx's bench CI step caught it | Fixed in cfgx's M11.2 PR; nothing for genx |
| 6 | Both examples | Each example needs the other package at a commit, not a tag: cfgx's at i18n's M6.1, i18n's at cfgx's M11.1 | As the roadmap planned; i18n tags `v0.2.0` first, then cfgx |

What went well: the declarations matched cfgx's old file on the first run in
both examples, `genx check` caught a stale labels file through cfgx's own
check, and nothing in either package's API had to change for genx.

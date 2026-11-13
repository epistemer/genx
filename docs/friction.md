# Friction log (M8.4)

What went wrong, or was awkward, when cfgx and i18n first generated through
genx (roadmap M8.2 and M8.3). Each entry ends as an amendment, a fix, or a
backlog item.

| # | Where | What happened | Outcome |
|---|-------|---------------|---------|
| 1 | i18n `examples/basic` under `go generate` | The directive sits in `package main`, so `go generate` set `GOPACKAGE=main`. genx passed it to cfgx, which takes its package name from `GOPACKAGE`, and cfgx wrote `zz_main.go` into `internal/settings` | Fixed in `v0.1.0-rc.3`: producers run without `go generate`'s variables (interface amendment A2) |

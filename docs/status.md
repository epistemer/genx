# Status log

Newest first. Every working session adds an entry: what landed, the decisions
taken for you, what's next, and what's waiting on you.

## 2026-11-13: the repository starts with its design records

- **Landed:** ADR 0001 (one generation pipeline over shared metadata) and the
  roadmap, moved here from cfgx's `docs/neozp/`, where they were parked
  while this repository didn't exist; an ADR template copied from cfgx; this
  log; a README. That is roadmap item M0.4 except `ownership.md`.
- **Changed in the move:** the name is no longer a working name, links to
  cfgx and i18n are absolute, and the roadmap drops the decision about the
  name and the repository.
- **Not reviewed yet:** ADR 0001 and the roadmap haven't had the
  cross-model review M0.5 asks for.
- **Next:** M0.1–M0.3 (module, CI, test harness), then the spikes.
- **Waiting on you:**
  - ownership: whether Claude owns genx until `v1.0.0` on cfgx's and i18n's
    terms;
  - cfgx PR #41 and i18n PR #7, which refer to these records;
  - a license. The roadmap plans MIT, like cfgx and i18n, in M0.1.

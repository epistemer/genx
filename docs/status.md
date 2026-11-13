# Status log

Newest first. Every working session adds an entry: what landed, the decisions
taken for you, what's next, and what's waiting on you. The terms are in
[ownership.md](ownership.md).

## 2026-11-13: ownership, M0 and the spikes; the interface waits for its freeze

- **Landed:**
  - **Ownership** ([ownership.md](ownership.md)): Claude owns genx until
    `v1.0.0`, on cfgx's and i18n's terms. MIT license.
  - **M0** ([PR #1](https://github.com/epistemer/genx/pull/1), green on all
    six jobs): the module and its package skeleton, CI, the testscript
    harness and the golden helper ([ADR 0002](adr/0002-test-harness.md)).
  - **M0.5** ([PR #2](https://github.com/epistemer/genx/pull/2)): the design
    review of ADR 0001 and the roadmap. 13 of 16 findings went into roadmap
    items; ADR 0001 gained changes 9–12 (inputs keyed by domain, generators
    declare `emits`, domain files encoded by genx, output path safety).
  - **M1.1–M1.5** ([PR #3](https://github.com/epistemer/genx/pull/3)): the
    five spikes, ADRs 0003–0007, and [ADR 0008](adr/0008-yaml.md). None hit
    its kill criterion. The labels pilot reproduced cfgx's 117 lines of
    declarations through gojq; sqlc's JSON request reproduced zp's
    `enums.go` (28 types, 112 values) with jq alone.
  - **M1.6, draft:** [interface.md](interface.md) and
    `schema/labels.v1.json`, revised by review 0003.
- **Reviewed:**
  - [0001](reviews/0001-gemini-m0-review.md), M0: Gemini 3.8 Flash
    (Medium), about 61k tokens; 6 of 15 findings accepted (most of the rest
    were disproved by green CI).
  - [0002](reviews/0002-gemini-design-review.md), ADR 0001 and the roadmap:
    Gemini 4 Hydrogen (High), about 39k.
  - [0003](reviews/0003-gemini-interface-review.md), the interface draft:
    Hydrogen (High), about 59k; all 19 findings taken, two in part.
  - The spikes ran on Gemini 3.8 Flash (High): S1 about 410k tokens, S2
    535k, S4 456k, S5 395k. I ran S3 myself and reproduced the claim each
    ADR rests on.
  - Argon was out of quota all session (about 42 hours left), so Hydrogen
    (High) took the design reviews. About 2.0M Gemini tokens in all.
- **Decided for you:**
  - The CLI has no command library; the scripts run `genx` through
    `testscript.Main`.
  - The spikes' code isn't kept: each ADR holds the evidence.
  - Under `check`, a producer without a check mode is trusted and named,
    not run in a copy (ADR 0007). This is in the freeze too.
- **Next:** once the interface is locked, M2 (the walking skeleton), then M3,
  M4 and M6 side by side.
- **Locked:** you locked interface.md the same day, choosing `use` by
  package, no `vars`, and `zz_` names for generator outputs. cfgx M11.1 and
  i18n M6.1 can start.
- **Waiting on you:**
  - cfgx PR #41 and i18n PR #7, which hold the design records. ADR 0003
    answers i18n's open question about where the generator's files sit.
  - Deleting the merged branches (`m0-foundations`, `m0.5-design-review`,
    `m1-spikes`), which the safety checks leave to you.
  - Your global CLAUDE.md still says every neozp generator emits its own
    i18n keys; ADR 0001 replaces that with "a generator writes a
    `labels/v1` file". I left that line for you to change.

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

# Status log

Newest first. Every working session adds an entry: what landed, the decisions
taken for you, what's next, and what's waiting on you. The terms are in
[ownership.md](ownership.md).

## 2026-11-13: M9, v0.1.0; i18n v0.2.0 and cfgx v0.2.0

- **Landed:**
  - **M9.1** ([PR #12](https://github.com/epistemer/genx/pull/12)):
    - [failures.md](failures.md) lists every failure with its message and the
      e2e script that shows it, and a test keeps each row true.
    - Messages that didn't say what to do next now do.
    - Ctrl-C stops the running step and writes nothing more. Before, genx
      died at once and could leave a temporary file behind.
  - **M9.3** ([PR #13](https://github.com/epistemer/genx/pull/13)):
    [review 0009](reviews/0009-gemini-m9.3-review.md) checked genxtest and
    both formats against the interface. All 14 findings were accepted: five
    fixes and ten missing tests.
  - **M9.2, M9.4, M9.5** ([PR #14](https://github.com/epistemer/genx/pull/14)):
    - the README's quickstart, which I followed step by step in a fresh
      `cfgx init` app;
    - the timings: cfgx's zp example takes 2.8 s warm, 2.5 s of it cfgx;
      zp's enums over its 5.4 MB sqlc request take 0.36 s;
    - the CHANGELOG.
  - **`v0.1.0`** is tagged and resolves through the module proxy.
  - **The release order is complete:**
    - i18n `v0.2.0` ([i18n PR #10](https://github.com/epistemer/i18n/pull/10)):
      genx finds its generator in an app that requires only i18n.
    - cfgx `v0.2.0` ([cfgx PR #42–#44](https://github.com/epistemer/cfgx/pull/44)):
      `App.Labels()`, with its example on the released versions of both.
- **Reviewed:** Gemini 4 Hydrogen (High), about 464k tokens, reviewed M9.3.
  Argon was out of quota. What it caught:
  - a YAML anchor used twice was refused as circular. The bug had also
    been the only guard against "billion laughs" files, so both YAML readers
    now have an expansion limit (amendment A3);
  - a broken second YAML document was ignored;
  - `genxtest.Render` accepted an input in another major version;
  - `use` accepted a `package main` directory.
- **Decided for you:**
  - Amendment A3, the alias limit: 10,000 values plus 100 per byte of the
    file. It only refuses files that no real app writes.
  - The timings don't call for parallel steps or caching, so backlog item
    8 stays where it is.
- **Next:** the backlog's first item, neozp's models package on sqlc. It is
  the second user the formats need before `v1.0.0`. On 2026-11-14 you
  created [epistemer/models](https://github.com/epistemer/models) and I
  prepared it for another session: its `docs/brief.md` holds what genx's
  work established, and spike S4's generators, checked again with `v0.1.0`.
  Its first open question comes back here: a generator can't be told an
  app's Go package or table prefix, which is backlog item "parameters for
  a generator in `use`" (interface §12).
- **Waiting on you:** nothing. `v1.0.0` is yours to call, after the models
  package has used the formats in a real app.

## 2026-11-13: M8, cfgx and i18n generate through genx; v0.1.0-rc.3

- **Landed:**
  - You merged cfgx PR #41 and i18n PR #7 and kept `genxtest`'s API as
    proposed; it's interface amendment A1.
  - **M8.2** (in i18n): the `labels` generator and its checks (i18n PR #8),
    then `examples/basic` through genx (i18n PR #9).
  - **M8.3** (in cfgx): `App.Labels()` (cfgx PR #42), then `examples/zp`
    through genx and `App.I18n()` removed (cfgx PR #43). The file genx
    writes has the declarations cfgx's own did; a test keeps it so.
  - **M8.4**: [friction.md](friction.md), six entries. The one genx fix:
    producers saw `go generate`'s `GOPACKAGE`, so cfgx wrote `zz_main.go`
    (amendment A2, [PR #10](https://github.com/epistemer/genx/pull/10),
    `v0.1.0-rc.3`). The generator guide gained a recipe for testing what a
    generator refuses ([PR #11](https://github.com/epistemer/genx/pull/11)).
- **Reviewed:** Gemini 3.8 Flash (High) reviewed i18n's generator (about
  228k tokens; 2 of 6 findings taken); Flash (Medium) reviewed cfgx M11.1
  (about 68k; 4 of 6).
- **Next:** M9, then `v0.1.0`; i18n `v0.2.0` and cfgx `v0.2.0` follow.
- **Waiting on you:** deleting the merged branches.

## 2026-11-13: M2–M7, the driver works; v0.1.0-rc.2

- **Landed:**
  - **M2** ([PR #4](https://github.com/epistemer/genx/pull/4)): the walking
    skeleton. Strict YAML for the manifest and metafiles, path rules, gojq
    with the refused list, templates, and a runner that writes only what
    changed, atomically, through `os.Root`.
  - **M3, M4** ([PR #5](https://github.com/epistemer/genx/pull/5)):
    everything a metafile can say, and the plan that orders producers and
    generators by the domains they need and write. `genx list`.
  - **M6** ([PR #6](https://github.com/epistemer/genx/pull/6)): generator
    packages from the module cache, `replace`, `go.work` and vendoring
    offline, tested through a local module proxy; generators not switched
    on; version mismatches that name the files.
  - **M5** ([PR #7](https://github.com/epistemer/genx/pull/7)):
    `genx.lock`, leftovers, never-replace, `genx check` (it writes nothing;
    a script compares the tree's hash), and `go generate`.
  - **M7** ([PR #8](https://github.com/epistemer/genx/pull/8)): `labels/v1`
    checks with 38 conformance files, `genxtest`, two guides, and the
    example module `examples/labels`, which CI tests.
  - **M8.1** ([PR #9](https://github.com/epistemer/genx/pull/9)):
    `v0.1.0-rc.2` is tagged; in an empty module `go get -tool` fetches it
    through proxy.golang.org and `go tool genx` runs. `v0.1.0-rc.1` can't be
    fetched: two conformance files differed only in case, which the go
    command refuses to zip. They're renamed, and a test now fails on any
    such pair.
  - **The `genx` skill**, `~/.claude/skills/genx/`, and a pointer to it in
    your global CLAUDE.md.
- **Reviewed:** reviews [0004](reviews/0004-gemini-m2-review.md) to
  [0008](reviews/0008-gemini-m7-review.md). Gemini wrote the leaf packages,
  the plan, M6 and the labels checks; I reviewed each and wrote the runner,
  M5 and genxtest, which Gemini reviewed. The catches that mattered:
  - the jq refusal walk named gojq's syntax tree by hand; it now walks
    every field by reflection;
  - on a first run a producer that needs a domain could run before the one
    that writes it;
  - `genx list` failed on a clean checkout, and a stale file on disk could
    raise a false version error;
  - a run that stopped before writing the lock would have wedged the next;
  - the example generator declared a name `labels/v1` doesn't allow.
- **Spent:** about 5.1M Gemini tokens over the whole session, nearly all on
  Gemini 3.8 Flash (High) (14 runs, 5.0M); Hydrogen (High) 98k for the two
  design reviews; Flash (Medium) 61k. Argon was out of quota throughout.
- **Decided for you:**
  - A producer's domains are read from its files before it runs when they
    exist, as a guess that orders steps but raises no error; on a first run
    they're learned as it runs. A step that needs a domain waits for
    producers that need nothing.
  - A file that already holds exactly what genx would write is taken as
    genx's, so a run that stopped before writing the lock recovers.
  - `genx list` lists with `-tags=genx`, so packages kept by ADR 0003's
    build-tagged blank import show up.
  - The guides' code is the example's files, checked by a test.
- **Next:** M8.2–M8.4 in i18n and cfgx, then M9 and `v0.1.0`.
- **Waiting on you:**
  - **genxtest's API** (interface §12): this session's summary has the
    question.
  - **cfgx PR #41 and i18n PR #7**, the design records. M8.2 (i18n's
    generator) and M8.3 (cfgx's `App.Labels()`) change both packages'
    APIs, so they wait for you to accept those records.
  - Deleting the merged branches and the broken tag `v0.1.0-rc.1`, which
    the safety checks leave to you.

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

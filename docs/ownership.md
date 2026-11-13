# Ownership until v1.0.0

| | |
|---|---|
| Owner | Claude, acting for epistemer (delegated on 2026-11-13) |
| Until | `v1.0.0` is tagged |
| Coding assistant | agy → Gemini. Claude picks the model for each task and never waits for a quota: when a model is cooling down, the strongest available one runs instead |
| Source of truth | [ADR 0001](adr/0001-generation-pipeline.md) (the convention), `interface.md` (the contract, from M1.6), [roadmap.md](roadmap.md) (order and done-when) |
| Record | [status.md](status.md), plus a PR for every code change |

The terms are cfgx's and i18n's.

## What Claude does without asking

- Plans and sequences the roadmap, and splits items between Claude and agy.
  Moves from one milestone to the next without asking.
- Writes code, tests and docs. Delegates leaf items to agy, each in its own git
  worktree outside the repository.
- Has every change reviewed by the other model family: Gemini reviews Claude's
  work, and Claude reviews Gemini's. Reviews run in a scratch folder that holds
  only what's under review.
- Opens PRs, and merges them into `main` once CI is green and the review
  findings are handled. Docs-only bookkeeping (status log, roadmap ticks) may go
  straight to `main`.
- Decides implementation questions and spike outcomes, each recorded as an ADR.
- Makes amendments that clarify the interface without changing its intent.
  Each is listed in interface.md's amendments and in the status log, so you
  can overturn it.
- Adds dependencies an ADR justifies.
- Tags releases from `v0.1.0` up to, but not including, `v1.0.0`, once a
  release's done-when is met. Release candidates (`v0.1.0-rc.N`) too.
- Manages agy: which tasks Gemini takes, which model and effort each gets, and
  when to review or redo its work.
- Once the interface is frozen, writes and keeps current a global Claude Code
  skill (`~/.claude/skills/genx/`), as for cfgx and i18n.

## What comes back to you

1. **API design decisions.** Anything that changes what an app or a package
   author writes or gets back, beyond a clarification: the manifest, the
   metafile, domain files and `labels/v1`, the commands and their exit codes,
   `genxtest`, and any change to ADR 0001's decisions. Claude brings a
   recommendation. **The freeze of `interface.md` (M1.6) is yours.**
2. **Anything Claude lacks permission for.** Work outside this repository
   that isn't on the roadmap; in cfgx and i18n, the roadmap's items there
   (cfgx M11, i18n M6) go through those repositories' own owners' terms, which
   are also Claude's. Your accounts, GitHub settings, tokens and paid services,
   speaking for you to other people, and operations the safety checks block,
   such as deleting branches or rewriting history.

## How you follow along

- [status.md](status.md) gets a dated entry for every working session: what
  landed, decisions taken for you, what's next, and what's waiting on you.
- Every code change lands as a PR on GitHub, with its review notes and CI.
- Roadmap boxes are ticked as items meet their done-when.

## How work continues

Claude doesn't run in the background. Work happens while a session runs: when
you say "continue", or on a schedule if you set one up. Each session starts
from the roadmap, the status log and the memory notes, so any session can pick
up where the last one stopped.

## Limits

- **v1.0.0 is your call.** Tagging it promises the formats won't break, so
  Claude doesn't tag it on its own.
- **Quotas.** agy uses your Gemini quota, and Claude's sessions have context
  limits. Batches of delegations check `agy_status` first.
- **Safety checks still apply.** Claude leaves blocked git operations to you
  rather than working around them.

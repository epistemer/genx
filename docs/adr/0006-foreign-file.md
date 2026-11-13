# ADR 0006: A foreign file, sqlc's request (spike S4)

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-13 |
| Roadmap item | M1.4 |
| Deciders | Claude, as owner. Gemini 3.8 Flash (High) ran the spike on a throwaway copy of zp (about 456k tokens); Claude checked the comparison |

## Context

A foreign tool knows nothing of domains. ADR 0001 lets a trivial generator
turn its output into a domain, and derive names with functions written in
jq. The question was whether that matches a real generator program's names,
on the second user the formats need before `v1.0.0`: zp's models on sqlc.

## What was tried

- **The request as JSON.** sqlc v1.29.0's own `gen: json:` output writes the
  codegen request, `models/codegen_request.json`, 5.4 MB for zp, with no
  database. Enums are at `.catalog.schemas[].enums[]`, each with `name` and
  `vals`.
- **Enums.** A 30-line query (with a `pascal` function) and a 37-line
  template wrote `models/enums.go`. Compared with zp's `enums.go` after
  gofmt (zp's committed file isn't gofmt'd): **identical**, 28 types, 112
  values, every constant, type, method and `SelectOption` list, and the 112
  i18n keys zp declares in the same file. zp builds and vets with it.
- **Labels.** A 29-line query wrote the enum values' labels as a
  `labels/v1` file (keys `enum.<type>.<value>`, text the value, a variable
  per value), so i18n's generator can take over the keys.

## What a query couldn't match

zp's enums are ASCII snake case with no initialisms, so `pascal` matched
everything. sqlc has rules a query can't see:

- `rename:` and the initialisms map from sqlc's Go options aren't in the
  request, so a generator can't honour them.
- sqlc resolves collisions statefully (`value_2`) and replaces some
  characters before casing.

## Decision

The kill criterion isn't met: for zp, the names match without sqlc's code.
Foreign files stay trivial-generator territory. The limit is recorded: when
an app uses sqlc's `rename` or initialisms, or values that collide, the
generator's names drift from sqlc's, and a test that pins the names (ADR
0001 §7) is what catches it. If the models package meets that case, it
needs a producer program, and the backlog item says so.

## Consequences

- The backlog's first item (the models package on sqlc) starts from a
  working query and template.
- The interface says a foreign file's names are derived by the generator,
  and that this is pinned by a test, not guaranteed.
- `pascal` uses `\p{L}` and `\p{N}` (ADR 0004).
- 5.4 MB is the largest input seen so far; S2's 2.6 MB took 0.2 s. M9.2
  times the real request.

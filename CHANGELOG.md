# Changelog

## v0.1.0 (2026-11-13)

The first release, built against the interface locked on 2026-11-13
([docs/interface.md](docs/interface.md)) and its amendments A1–A3.

- **The manifest**, `genx.yaml`, at the app's module root: producers (a
  command, its `dir`, an optional `check` command, a `timeout`, the domains it
  `needs`, and the domain files it `emits`, foreign files with their domain),
  and `use`, which switches on generators by Go package import path.
- **Trivial generators**, a `genx.meta.yaml` in an importable package's
  directory: jq queries (gojq v0.12.20, with the clock, the environment and
  file reads refused) for the units and each unit's output path, and either a
  Go `text/template` or `emits`, which writes the unit as a domain file for
  later generators. Packages resolve through the go command: the module
  cache, `replace`, `go.work` and vendoring work as they do for builds.
- **Running:** `genx` orders every step by the domains it needs and writes,
  runs it, and writes only the generated files whose bytes changed,
  atomically and only inside the module. It deletes files nothing produces
  any more, never replaces a file it didn't write, and keeps a Go file that
  lost its generated-code header. `genx.lock` lists what it wrote.
- **`genx check`** writes nothing and reports every stale, missing or
  leftover file, and every producer whose own check fails. **`genx list`**
  shows the steps in order, and the generators in the build that `use`
  doesn't switch on.
- **`labels/v1`**, the first shared domain: a JSON Schema, checks beyond it
  on every labels file genx reads, and 44 conformance files for producers'
  tests: 6 valid, and 38 invalid, each with its message.
- **`genxtest`**: `AssertGenerated` runs `genx check` from a test, and
  `Render` runs one generator over files, for golden tests.
- **Messages** name the file and line, the step or the producer, and say
  what to do next; [docs/failures.md](docs/failures.md) lists each with the
  test that shows it. The first Ctrl-C stops the running step and writes
  nothing more.
- **Guides** for writing a producer and a generator, and a complete example
  generator in `examples/labels`.

cfgx (`App.Labels`, from v0.2.0) and i18n (the `labels` generator, from
v0.2.0) are its first users.

### Since v0.1.0-rc.3

- `use` refuses a `package main` directory, and resolves packages with
  `-tags=genx`, as `genx list` lists them.
- A YAML anchor reached on two paths is no longer refused as circular, and a
  file whose aliases expand past a limit is refused (amendment A3). A broken
  second document in a manifest or metafile is an error.
- `genxtest.Render` refuses an input in another major version than the
  generator needs.
- Non-Go outputs get LF line endings, whatever a unit's text holds.
- Clearer messages for a missing command, a timeout, a domain nothing
  writes, a package without a metafile or a template, names and domains in
  the wrong form, and a damaged `genx.lock`.

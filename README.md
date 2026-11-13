# genx

The code generation driver for neozp. Packages publish data about what they
generate (domain files), other packages ship generators that are only data
(queries and templates), and genx runs both in order, checks that generated
files are current, and removes what nothing produces any more.

Nothing is built yet. The design and the plan:

- [docs/adr/0001-generation-pipeline.md](docs/adr/0001-generation-pipeline.md):
  why a driver, and the convention it runs.
- [docs/roadmap.md](docs/roadmap.md): the order of work to `v0.1.0`.
- [docs/status.md](docs/status.md): the log of working sessions.

Part of neozp, alongside [cfgx](https://github.com/epistemer/cfgx) and
[i18n](https://github.com/epistemer/i18n).

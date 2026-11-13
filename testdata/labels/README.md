# labels/v1 Conformance Files

These files are the conformance test suite for `labels/v1` documents specified in `docs/interface.md` §11 and `schema/labels.v1.json`.

- `valid/*.json`: Valid `labels/v1` documents demonstrating minimal, full with every optional field, root package dir ".", unknown extra fields, and empty text cases.
- `invalid/*.json`: Documents violating exactly one validation rule, each paired with a sibling `.err` file containing the exact error message from `internal/labels.Check`.

Producers (such as cfgx in roadmap M11.1) copy these conformance files into their own test suites. A producer's own labels files must pass every valid case's rules.

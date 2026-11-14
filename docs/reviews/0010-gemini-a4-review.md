# Review 0010: amendment A4, parameters for a generator

| | |
|---|---|
| Date | 2026-11-14 |
| Reviewer | Gemini 4 Hydrogen (High), about 82k tokens (Argon was out of quota). Claude checked each finding against the code before fixing it |
| Scope | The diff of A4 against `main`, judged against A4's text in interface.md §13. The review ran in a scratch folder that held only the diff and that text |

No finding was a divergence from the approved design.

## Findings

| # | Finding | Verdict |
|---|---------|---------|
| 1 | `RenderParams` normalized values in map order and returned on the first that JSON can't encode, so which one it named could vary, and other problems went unreported | Accepted: values are read in name order, and every problem is reported at once, as genx does for a use entry |
| 2 | No test ran one package in two `use` entries with different values, `params: {}`, an object or array value in `genx list`, `$params` in a metafile that declares none, or aliases and merge keys in values | Accepted: `params_entries.txtar`, and cases in `internal/manifest` |
| 3 | `DecodeNode` gives each value a budget of 10,000 alias expansions instead of the file's | Rejected: yamlx expands the file's aliases within the file's budget (A3) before any value is decoded, so `DecodeNode` never meets an alias. A comment now says so |
| 4 | In the map form, `params` went unchecked when `generators` was missing, and an empty or repeated generator was reported at the entry instead of at `generators` | Accepted, both |
| 5 | `query.Kind` called any Go type it didn't know a number | Accepted: only decoded numbers are; anything else names its Go type, which only a bug in genx can produce |
| 6 | A bad value nested in an object or list parameter was reported at the line where the value starts | Accepted: the message has the line of the node that failed |

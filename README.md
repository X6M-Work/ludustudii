# Backend linting

Run these commands from the repository root. The devcontainer installs the pinned
standalone golangci-lint binary; lint tooling is kept out of `go.mod`.

| Command | Purpose |
| --- | --- |
| `make lint` | Check the backend and tagged tests. |
| `make lint-fix` | Apply available automatic lint fixes. |
| `make lint-fmt` | Format Go source with gofumpt and goimports. |
| `make lint-verify` | Validate `backend/.golangci.yml`. |

Lint uses the `unit,integration,apidocs` build tags so tests are included.
The alternate `script_none.go` build is intentionally omitted from linting.

## Exclusion policy

Use `backend/.golangci.yml` for shared rules: generated files and bundled assets,
standard exclusion presets, test fixtures, and package boundaries.

For a one-off exception, place `//nolint:<linter> // <reason>` beside the code
with the smallest practical scope. Name the linter and explain the deliberate
behavior or false positive; prefer a real fix when the finding identifies a bug.
For gosec, use `// #nosec Gxxx -- <reason>` to suppress only the relevant check.

## Deliberately disabled linters

These linters are omitted from the standard set plus the explicit enable list.

| Linter | Reason |
| --- | --- |
| `wrapcheck` | Stores may return raw Scan/Query errors; handlers own logging and context. |
| `testpackage` | Handler tests stay in-package to replace bind/sanitize/validate function-field seams. |
| `gochecknoglobals` | Package-level domain values in `internal/services/lcards/domain/global.go` are intentional. |
| `exhaustruct` | Partial struct literals and zero-value defaults are intentional; naming every field adds noise. |
| `varnamelen` | Short idiomatic names are useful when their scope and meaning are clear. |
| `wsl` | Blank-line layout is left to readability and the configured formatters. |
| `nlreturn` | A blank line before every return is not a project requirement. |
| `ireturn` | Interface returns are allowed where they express the intended abstraction. |
| `mnd` | Literals need names when they convey domain meaning, not solely because they are numbers. |
| `lll` | Line wrapping follows readability rather than a fixed column limit. |
| `funlen` | Function length alone does not justify splitting cohesive code or table-driven tests. |
| `gocognit` | Review control flow for clarity rather than enforcing a numeric complexity threshold. |
| `paralleltest` | Parallel execution is chosen per test; `tparallel` checks consistency where it is used. |

# Ludustudii
Ludus Studii ("The Study Game") — Turn complex study material into structured, gamified learning loops.

## Development
### Backend linting

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

#### Exclusion policy

Use `backend/.golangci.yml` for shared rules: generated files and bundled assets,
standard exclusion presets, test fixtures, and package boundaries.

For a one-off exception, place `//nolint:<linter> // <reason>` beside the code
with the smallest practical scope. Name the linter and explain the deliberate
behavior or false positive; prefer a real fix when the finding identifies a bug.
For gosec, use `// #nosec Gxxx -- <reason>` to suppress only the relevant check.

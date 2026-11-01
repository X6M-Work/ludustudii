# Backend lint setup

Pinned golangci-lint **v2.15.0**, installed as an official standalone binary by
`.devcontainer/post-create.sh`, outside go.mod. `golangci-lint version` reports
**go1.27.0**; the user explicitly approved this exception to the requested
Go >= 1.27.3 build toolchain. The backend itself uses Go 1.27.3.

Settings were checked against `golangci-lint linters` and the
[v2.15.0 configuration reference](https://github.com/golangci/golangci-lint/blob/v2.15.0/.golangci.reference.yml).

Run `make lint`, `make lint-fix`, `make lint-fmt`, or `make lint-verify`
from the repository root.
Lint runs once with unit,integration,apidocs. The alternate script_none.go build
is intentionally omitted from linting.
There are no local-prefixes or gci import groups.

## Findings

Counts below are unique baseline diagnostics across both build variants (115 total),
with issue limits and line deduplication disabled. Both variants had the same findings.
Current lint output is zero issues with the configured build tags.

| Linter / formatter | Baseline | Resolution |
| --- | ---: | --- |
| errcheck | 7 | Check HTTP/database close errors; make AssertPgCode a void assertion with a private error-returning helper; standard exclusions allow idiomatic deferred rows.Close calls. |
| govet | 0 | No findings. |
| ineffassign | 0 | No findings. |
| staticcheck | 3 | Replace three identical-field struct literals with conversions. |
| unused | 0 | No findings. |
| errorlint | 3 | Explained inline directives preserve the depth-first error-tree walker and exact constructor error identity test. |
| rowserrcheck | 0 | No findings. |
| sqlclosecheck | 0 | No findings. |
| bodyclose | 47 | Scoped exclusions for testkit.Do calls: the helper already consumes and closes the response body. |
| noctx | 21 | Use httptest.NewRequestWithContext(t.Context(), ...) in tests. |
| usetesting | 7 | Use t.Context() in logging tests. |
| contextcheck | 2 | Derive cleanup contexts with context.WithoutCancel; use a private context-taking close helper without changing the exported API. |
| exhaustive | 1 | Allow default cases to satisfy exhaustiveness; the reflective walker ignores leaf kinds with default. |
| nilerr | 0 | No findings. |
| nilnil | 0 | No findings. |
| gosec | 3 | Targeted inline #nosec directives cover intentional CLI artifact writing and an environment-name false positive. |
| revive | 1 | Document the exported logging format constant block. |
| godot | 0 | No findings. |
| sloglint | 15 | Use slog.DiscardHandler; preserve environment variable names as log keys with an explained local exception. |
| depguard | 0 | No findings. |
| forbidigo | 0 | No findings. |
| thelper | 2 | Name testing.TB parameters tb; parameter and result types are unchanged. |
| tparallel | 0 | No findings. |
| nolintlint | 0 | No findings. |
| usestdlibvars | 0 | No findings. |
| copyloopvar | 0 | No findings. |
| intrange | 0 | No findings. |
| unconvert | 0 | No findings. |
| misspell | 0 | No findings. |
| errname | 0 | No findings. |
| perfsprint | 1 | Concatenate the CREATE DATABASE prefix. |
| modernize | 1 | Range over strings.SplitSeq. |
| unparam | 0 | No findings. |
| gocritic | 0 | No findings. |
| gofumpt | 1 | Group adjacent constant declarations. |
| goimports | 0 | No findings. |

The trial linters **unparam** and **gocritic** both reported zero findings and remain enabled.
The intentionally excluded linters requested by the user remain disabled.

## Exclusions and deliberate behavior

- `exhaustive.default-signifies-exhaustive: true` lets an explicit default case
  cover remaining enum values, including leaf kinds in the reflective test walker.

- Enable `std-error-handling` and `common-false-positives` presets for standard
  cleanup patterns and known false positives, including deferred `rows.Close()`.

- Both linter and formatter paths exclude `internal/infra/apidocs/dist` and
  `internal/infra/spa/dist`: bundled assets, not maintained Go source.
- Generated files use `lax` exclusion for both lint and formatting.
- `_test.go`: gosec is excluded for security fixtures. noctx remains enabled;
  its 21 findings were fixed.
- Environment access: forbidigo permits only `os.Getenv` / `os.LookupEnv`
  in `internal/infra/env` and `internal/testkit`. fmt.Print* and legacy log API
  prohibitions remain active there.
- bodyclose: only test sources calling `testkit.Do`, plus the bare `Do` call in
  `internal/testkit/openapi_test.go`; Do reads and closes the body before returning.
- Inline `nolint:sloglint` on `env.Vars.LogAttrs`: keep
  original environment variable names as log keys. Other logging uses snake_case.
- Inline `nolint:errorlint`: only assertion/switch diagnostics in `internal/kit/erresp/error.go`;
  this code walks every branch itself, whereas errors.As would skip or duplicate
  siblings. Also the `got.Err != err` constructor test in
  `internal/kit/validator/errors_test.go`, which pins exact error identity.
- Inline `#nosec G306 G703`: only `cmd/tools/bundlespec/main.go`; the local generator
  intentionally writes a public specification with mode 0644 to its CLI output path.
- Inline `#nosec G101`: only the `EnvAppWriteTimeout = "APP_WRITE_TIMEOUT"` declaration
  in `internal/infra/env/load.go`; an environment variable name is not a credential.
- depguard excludes test files from its architecture rules, allowing in-package
  handler tests to use their existing seams and test helpers. Both root and nested
  kit/services files are covered. Production imports of testkit are forbidden.

Source-specific exceptions use explained inline directives; shared policies stay in the
YAML exclusions. AssertPgCode now returns nothing; AssertPgConstraint uses a private
helper for the underlying PostgreSQL error, as requested.
Stores, in-package tests, and intentional package-level domain values retain their design.
`rdb.Connect` did not produce a contextcheck finding; its API was left as-is.
No further decisions are required.

## Verification

- `make lint-verify` (`golangci-lint config verify`) passed.
- `make lint-fmt` applied formatting; `golangci-lint fmt --diff` was empty.
- `make lint` passes with the configured unit,integration,apidocs build tags.
- `make test-unit` passed after each fix tier, including both apidocs variants.
- `make test-integration` passed after each fix tier against the available PostgreSQL
  test database, with the target's race detector and uncached execution.
- Isolated synthetic packages verified all seven dependency prohibitions, allowed
  test imports, aliased legacy log detection, and environment access exceptions.

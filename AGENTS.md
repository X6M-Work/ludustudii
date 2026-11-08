# AGENTS.md

Use `README.md` for the stack, layout, API conventions, configuration and Make
targets. Before changing an area, read its relevant section. This file adds the
repository-specific guidance that helps you make changes consistently.

Run Make commands from the repository root. Unless prefixed with `backend/`,
backend source paths and `PKG` values below are relative to `backend/`.

## Definition of done

- Make sure `make check` passes before finishing. It needs `DB_TEST_CONNECTION` (set in the
  devcontainer).
- CI runs `make check` on every push and pull request (`.github/workflows/ci.yml`).
  A failing CI run means the change isn't done.
- While iterating, use focused tests, for example:
  `make test-unit PKG=./internal/services/lcards/... RUN='TestHandler_Handle'`.
  Use the same `PKG` and `RUN` options with `test-integration`.
- Keep the OpenAPI spec, code and tests together in the same change.

## Git workflow

- Never commit to `main` directly. Create a branch with a short, descriptive
  name (for example `feat/decks`), commit there, and push the branch.
- After pushing, open a pull request against `main`
  (`gh pr create --base main`) with a title in the form `type: summary`
  (types: `feat`, `fix`, `refactor`, `test`, `docs`, `build`, `ci`, `dev`)
  and a body that follows `.github/pull_request_template.md`
  (`gh pr create --body` skips the template). Then stop.
- CI (`.github/workflows/pr-title.yml`) enforces the title format: a listed
  type, no scope, one space after the colon, a summary starting with a
  lowercase letter or a digit, no trailing period, and at most 64 characters
  so the squash-merge subject with ` (#N)` stays within 72. Check a title
  locally with `PR_TITLE='type: summary' make pr-title-check`.
- Fixes requested in review go on the same branch as new commits; the pull
  request updates automatically.
- Never merge, approve, enable auto-merge, or rebase onto `main` and
  force-push. The maintainer reviews and squash-merges.
- Commits are authored by the maintainer only. Don't add `Co-authored-by`
  trailers, "Generated with" lines, or any other agent attribution to commit
  messages or pull request text.

## Test file naming

- Unit tests use `<name>_test.go` with `//go:build unit`.
- Integration tests use `<name>_integration_test.go` with
  `//go:build integration`.
- Service API tests are the exception: keep `api_test.go` with
  `//go:build integration`.

Keep the build tags: filenames identify the test scope, while tags select
the suite that Make runs. `make check` fails when a name and tag disagree.

## Adding an endpoint

Follow the structure in `internal/services/lcards`, using `lcards/create` as
the reference slice.

1. Spec first: add or extend a file in `backend/apispec/paths/`, reusing
   `components/` for schemas, parameters and responses. Run `make spec-bundle`.
2. Schema changes: `make migrate-create NAME=<verb_noun>`. Write both `Up` and
   `Down`. Mirror input limits as `CHECK` constraints.
3. Shared rules, limits and constants go in `<svc>/domain/`.
4. One subpackage per endpoint with `handler.go` and `store.go`. The handler
   binds, sanitizes, validates, then calls the store through a small interface.
   Each store query lives in `<slice>/<name>.sql` with a placeholder header comment
   and is embedded with `//go:embed`; no inline SQL strings in stores.
5. Register each route in `<svc>/routes.go` with
   `api.HandleFunc("METHOD path", h.Handle)` on the `*httpkit.API` it receives,
   one call per method and path. Always include the method; GET also serves
   HEAD. The API turns the mux's 404 and 405 (with its `Allow`) into the JSON
   error shape. A new service registers in `internal/app/api/api.go`.
6. Test each layer using the existing patterns:
   - `handler_test.go` (`//go:build unit`): isolate binding, validation and store
     behavior using the existing function fields and interfaces.
   - `store_integration_test.go` (`//go:build integration`): real SQL against a `testkit`
     database. No store mocks or store unit tests.
   - `api_test.go` in the service package (`//go:build integration`): mount the
     API surface alone (`router.New(log, onError, api.New(db, onError))`) and
     drive it through `testkit.Do`, which validates every response against the spec.
   - A package with integration tests needs `TestMain` calling `testkit.Run`.
   - When changing validation or normalization rules, extend the domain unit
     tests and sanitizer/validator fuzz contracts in `<svc>/contract/`.

## Composing the server

`internal/app` is the composition root. HTTP surfaces (probes, API, API
reference, client app) are built independently and mounted on the shared
router in `internal/app/handler.go`. Keep each surface free of the others'
dependencies. A surface takes its own parts and the error handler, never the
whole configuration. Process lifecycle changes (signals, draining, shutdown)
go in `internal/app/server`, and shared middleware goes in
`internal/app/router`. Neither may know a surface. See README.md (Backend →
Surfaces and startup).

## Rules not enforced by tooling

- Every UPDATE/DELETE on a user-editable table requires `If-Match` and checks
  the expected `version` in its SQL predicate. UPDATE also bumps `version` in
  the same statement; DELETE removes the matching row. Follow `lcards/update`
  as the reference pattern.
- Every table has `created_at` and `updated_at` (`timestamptz NOT NULL`, no
  default). INSERT sets both explicitly to `now()`. Every UPDATE sets
  `updated_at = now()` alongside the `version` bump. Never use `time.Now()`
  for row timestamps, and never write `created_at` after insert.
- Prefer the standard library. Ask before adding a Go dependency.
- Write SQL by hand, without an ORM or query builder.
- Errors reach clients only through the shared error shape in `README.md`
  (Backend → API, error response format and codes). Add any new error code to
  that table and to the spec.
- Regenerate `internal/infra/apidocs/dist/openapi.json` with `make spec-bundle`
  rather than editing it by hand.
- Add a new migration when the schema changes. Squashing migrations belonging
  to one feature or editing existing migrations is permitted only after asking
  the user and receiving approval. Agents never do either on their own; they
  add a new migration. See [ADR 0017](docs/adr/0017-migration-policy.md).
- Keep the shared lint rules in `backend/.golangci.yml` intact. Follow
  `README.md` (Linting → Exclusion policy) for one-off exceptions.

## Decisions

Architecture decisions are recorded in `docs/adr/` (index:
`docs/adr/README.md`). Before changing an area, check whether a decision
record covers it. Follow accepted decisions. If one seems wrong, stop and
propose an amending or superseding record instead of changing the code to
contradict it. `docs/adr/archive/` is history only: ignore it and never follow
an archived record.

## Checks handled by tooling

Package boundaries (depguard), env access only in `internal/infra/env`, `slog`
only for logging, formatting, tidy modules, a fresh spec bundle and test file
names matching their build tags are all checked by `make check`. Resolve
findings in the code while preserving these checks.

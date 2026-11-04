# AGENTS.md

Use `README.md` for the stack, layout, API conventions, configuration and Make
targets. Before changing an area, read its relevant section. This file adds the
repository-specific guidance that helps you make changes consistently.

Run Make commands from the repository root. Unless prefixed with `backend/`,
backend source paths and `PKG` values below are relative to `backend/`.

## Definition of done

- Make sure `make check` passes before finishing. It needs `DB_TEST_CONNECTION` (set in the
  devcontainer).
- While iterating, use focused tests, for example:
  `make test-unit PKG=./internal/services/lcards/... RUN='TestHandler_Handle'`.
  Use the same `PKG` and `RUN` options with `test-integration`.
- Keep the OpenAPI spec, code and tests together in the same change.

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
5. Register each route in `<svc>/routes.go` with
   `api.HandleFunc("METHOD path", h.Handle)` on the `*httpkit.API` it receives,
   one call per method and path. Always include the method; GET also serves
   HEAD. The API turns the mux's 404 and 405 (with its `Allow`) into the JSON
   error shape. A new service registers in `internal/app/router.go`.
6. Test each layer using the existing patterns:
   - `handler_test.go` (`//go:build unit`): isolate binding, validation and store
     behavior using the existing function fields and interfaces.
   - `store_test.go` (`//go:build integration`): real SQL against a `testkit`
     database. No store mocks or store unit tests.
   - `api_test.go` in the service package (`//go:build integration`): drive the
     router through `testkit.Do`, which validates every response against the spec.
   - A package with integration tests needs `TestMain` calling `testkit.Run`.
   - When changing validation or normalization rules, extend the domain unit
     tests and sanitizer/validator fuzz contracts in `<svc>/contract/`.

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
- Add a new migration when the schema changes; leave applied migrations intact.
- Keep the shared lint rules in `backend/.golangci.yml` intact. Follow
  `README.md` (Linting → Exclusion policy) for one-off exceptions.

## Checks handled by tooling

Package boundaries (depguard), env access only in `internal/infra/env`, `slog`
only for logging, formatting, tidy modules and a fresh spec bundle are all
checked by `make check`. Resolve findings in the code while preserving these
checks.

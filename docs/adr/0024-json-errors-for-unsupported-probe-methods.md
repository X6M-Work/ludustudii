# 0024. JSON errors for unsupported probe methods

- Status: Accepted
- Date: 2026-11-06
- Tags: back-end, api, operations

## Context

ADR 0023 gives unsupported probe methods a plain-text 405 response. Other
non-file errors use the shared JSON error shape and central error handler.
The probe exception makes clients handle an additional error format.
ADR 0007 limits that shape to API paths and assigns plain-text errors to
non-API paths, which includes the operational probes.

## Decision

This record amends [ADR 0023](0023-liveness-and-readiness-probes.md) only for
unsupported probe methods, and
[ADR 0007](0007-one-json-error-shape-with-machine-readable-codes.md) only in
its API-only scope for the shared JSON error shape. The non-API paths
`/livez` and `/readyz` also use that shape for method errors and shared
middleware errors. Their GET and HEAD probe status responses retain their
existing contracts. ADR 0007 and ADR 0023 remain Accepted; their other rules
continue to apply.

- Methods other than GET and HEAD on `/livez` and `/readyz` return 405 with
  `Content-Type: application/json` and `{"code":"method_not_allowed"}`.
- Probes pass the error to the shared error handler supplied by the app.
- These responses retain `Allow: GET, HEAD`, `Cache-Control: no-store` and
  `X-Request-Id`. Probe request summaries continue to use debug level.
- GET and HEAD probe status responses retain their existing contracts.

## Alternatives considered

- **Keep plain-text 405 responses.** Preserves the exception but requires a
  second error format for clients of non-file endpoints.
- **Write the JSON error inside health.** Duplicates the central handler's
  response encoding and error logging behavior.

## Consequences

Clients expecting a plain-text probe method error must handle the shared JSON
shape instead. The OpenAPI path descriptions, response validator and router
tests pin the response body and retained headers.

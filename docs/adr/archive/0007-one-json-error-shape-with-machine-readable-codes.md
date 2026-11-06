# 0007. One JSON error shape with machine-readable codes

- Status: Superseded by [0025](../0025-error-responses.md)
- Date: 2026-11-05
- Tags: back-end, front-end, api

Amended by [0024](0024-json-errors-for-unsupported-probe-methods.md) only in
the API-only scope of the shared JSON error shape: method errors and shared
middleware errors on the non-API paths `/livez` and `/readyz` also use it.
All other rules below continue to apply.

## Context

Clients need to tell errors apart reliably, and the client app has to show
messages in more than one language. Human-readable server messages tie
wording and language to the server and invite clients to parse text.

## Decision

Every API error response has the same JSON body:

```json
{
  "code": "validation_error",
  "errors": [{ "field": "title", "code": "max_length" }]
}
```

- `code` is a stable, machine-readable identifier for the error. Clients
  branch on it, together with the HTTP status.
- `errors` lists per-field failures and is present only for validation
  errors. Each entry names the field (body field, path parameter or query
  parameter) and a field-level code.
- **No human-readable messages.** The client owns all wording and
  translation. Server-side detail goes to logs only.
- Every code is documented in the spec and in the README's error table.
  Adding a code is an API change.
- The shape applies to the API only. Non-API paths (the client app) answer
  with plain text.

## Alternatives considered

- **RFC 9457 Problem Details** (`application/problem+json`). A standard, but
  its main fields (`type`, `title`, `detail`) are human-oriented or need URI
  identifiers the project has no use for. The useful part is a code and field
  errors, which this shape covers more simply.
- **Messages in the response.** Easier for quick debugging, but couples
  wording and language to the server and leaks into client logic.
- **Status code only.** Not enough to distinguish errors that share a status,
  such as different 400s.

## Consequences

- Clients handle errors with a lookup on codes, and localization stays in the
  client.
- Codes are part of the public contract: renaming or removing one is a
  breaking change.
- Error responses are harder to read by eye during debugging. Logs and
  request IDs fill the gap.

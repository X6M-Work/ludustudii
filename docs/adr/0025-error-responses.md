# 0025. Error responses

- Status: Accepted
- Date: 2026-11-06
- Tags: back-end, front-end, api
- Consolidates: [0007](archive/0007-one-json-error-shape-with-machine-readable-codes.md), [0024](archive/0024-json-errors-for-unsupported-probe-methods.md)

## Context

Clients need to tell errors apart reliably, and the client app has to show
messages in more than one language. Human-readable server messages tie
wording and language to the server and invite clients to parse text.

Errors come from several layers: API handlers, the API router for unknown
paths and methods, the operational probes, and shared middleware that runs
before any route is chosen. Each additional format is one more thing every
client has to handle.

## Decision

Error responses use one JSON body:

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
- Every code is documented in the OpenAPI spec and in the README's error
  table. Adding a code is an API change.
- Errors are written by the central error handler
  ([0006](0006-central-error-handling.md)), never encoded by hand elsewhere.

The shape applies to:

- **All API errors**, including unknown API paths and unsupported methods.
- **Method errors on the probe paths.** Successful and unavailable probe
  responses keep their own status contract.
- **Errors produced by shared middleware on every path**, client-app paths
  included, such as oversized request bodies and recovered panics.

Errors produced by the client app's own handler (unsupported methods, missing
static files, file system failures) are plain text.

## Alternatives considered

- **RFC 9457 Problem Details** (`application/problem+json`). A standard, but
  its main fields (`type`, `title`, `detail`) are human-oriented or need URI
  identifiers the project has no use for. The useful part is a code and field
  errors, which this shape covers more simply.
- **Messages in the response.** Easier for quick debugging, but couples
  wording and language to the server and leaks into client logic.
- **Status code only.** Not enough to distinguish errors that share a status,
  such as different 400s.
- **Plain-text method errors on the probes.** Keeps the probes independent of
  the error handler, but gives non-browser clients a second error format.
- **Plain-text middleware errors on client-app paths.** Middleware runs
  before routing and cannot know which handler a request would reach; choosing
  a format there would duplicate routing decisions for little benefit.

## Consequences

- Clients handle errors with a lookup on codes, and localization stays in the
  client.
- Codes are part of the public contract: renaming or removing one is a
  breaking change.
- Error responses are harder to read by eye during debugging. Logs and
  request IDs fill the gap.
- A client-app path can answer with either format: plain text from the client
  app's handler, or JSON from shared middleware. Browsers rarely see the
  middleware errors, so the mix is accepted.
- The shared test harness validates every API and probe response it drives
  against the spec, including error bodies. Router tests pin the client app's
  plain-text errors. The JSON format of middleware errors on client-app paths
  is enforced only by this record and the shared middleware chain.

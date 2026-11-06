# 0026. Request IDs and request logging

- Status: Accepted
- Date: 2026-11-06
- Tags: back-end, api, observability
- Consolidates: [0012](archive/0012-request-ids-and-one-log-record-per-request.md), [0021](archive/0021-bounded-client-request-ids.md), [0023](archive/0023-liveness-and-readiness-probes.md)

## Context

When something fails, we need every log line for that request and a way to
match it to what the client saw. Logging from many places without a shared
identifier makes that guesswork, and per-step access logs are noisy.

A client app or proxy may already have assigned a correlation ID. That value
is client-controlled, though, and ends up in response headers and in every
log record for the request. Some paths, such as operational probes, are
requested so often that their summaries would bury everything else.

## Decision

- **Every request has an ID.** The server reuses the first incoming
  `X-Request-Id` only when it is short and limited to a safe character set;
  otherwise it generates a random ID. The exact limits are defined in the
  OpenAPI spec and the README.
- **Client values are never altered or rejected.** They are not trimmed,
  truncated or normalized; an unusable value is replaced. A missing or
  invalid ID never causes an error response. The incoming request header is
  left unchanged.
- **The selected ID is returned** in the `X-Request-Id` header of every
  response.
- **The ID is carried in the request context** and added automatically to
  every log record written with that context.
- **One summary record per request** is written after the final response,
  including responses produced by errors and recovered panics. It holds the
  method, the matched route pattern, the path, the status and the latency.
- **Its level follows the status:** error for 5xx, warning for 4xx, info
  otherwise.
- **Quiet paths** are an explicit set of exact paths passed to the request
  logger. Their summaries log at debug at every status. Paths are made quiet
  only when they are polled frequently and their state changes are logged
  elsewhere, as the probes do ([0027](0027-liveness-and-readiness-probes.md)).

## Alternatives considered

- **Always generate the ID and ignore the client's.** Simpler, but loses the
  link to IDs set by the client app or a proxy in front of the server.
- **Accept client IDs as-is.** Lets one client-controlled header carry
  arbitrary content and inflate many log records.
- **Reject requests with an invalid ID.** Makes an optional correlation header
  prevent useful work and adds a client-visible validation error.
- **Truncate or sanitize the ID.** Changes correlation values and can collapse
  distinct client IDs into the same value.
- **W3C Trace Context and distributed tracing** (`traceparent`, OpenTelemetry).
  The right tool once requests cross services; premature for a single binary.
- **Separate start and end records**, or logging from each middleware.
  Doubles the volume and splits the information for one request.
- **Status-based levels for every path.** Frequent probes during an outage
  would flood warnings and errors and obscure the state change itself.

## Consequences

- One ID connects the client's error report, the response and every log line.
- The route pattern groups requests by endpoint regardless of IDs in the
  path, which keeps log queries and statistics simple.
- Client-controlled IDs have bounded size and a predictable character set.
  Common UUIDs and proxy correlation IDs remain usable; clients sending other
  values receive a generated ID and must read it from the response header.
- Failures on quiet paths are invisible at normal log levels, so each quiet
  path needs its own transition logging.
- When services are split, this will need to move to standard trace
  propagation.
- Middleware and router tests cover ID selection, the response header, log
  attributes and levels; the shared test harness checks the header against
  the spec.

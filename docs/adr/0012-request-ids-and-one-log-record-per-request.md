# 0012. Request IDs and one log record per request

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, api, observability

## Context

When something fails, we need every log line for that request and a way to
match it to what the client saw. Logging from many places without a shared
identifier makes that guesswork, and per-step access logs are noisy.

## Decision

- **Every request has an ID.** The server reuses the client's `X-Request-Id`
  if present, and otherwise generates a random one. The ID is returned in the
  response's `X-Request-Id` header.
- **The ID is carried in the request context** and added automatically to
  every log record written with that context.
- **One summary record per request** is written after the final response,
  including responses produced by errors and recovered panics. It holds the
  method, the matched route pattern, the path, the status and the latency.
- **Its level follows the status:** error for 5xx, warning for 4xx, info
  otherwise.

## Alternatives considered

- **Always generate the ID and ignore the client's.** Simpler, but loses the
  link to IDs set by the client app or a proxy in front of the server.
- **W3C Trace Context and distributed tracing** (`traceparent`, OpenTelemetry).
  The right tool once requests cross services; premature for a single binary.
- **Separate start and end records**, or logging from each middleware.
  Doubles the volume and splits the information for one request.

## Consequences

- One ID connects the client's error report, the response and every log line.
- The route pattern groups requests by endpoint regardless of IDs in the
  path, which keeps log queries and statistics simple.
- Client-supplied IDs are used as-is. Their content isn't validated, and
  their length is bounded only by the server's header size limit.
- When services are split, this will need to move to standard trace
  propagation.
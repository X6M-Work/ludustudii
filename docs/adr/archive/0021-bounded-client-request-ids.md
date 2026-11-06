# 0021. Bounded client request IDs

- Status: Superseded by [0026](../0026-request-ids-and-request-logging.md)
- Date: 2026-11-05
- Tags: back-end, api, observability

## Context

ADR 0012 accepts client-supplied request IDs without validation. Those values
are echoed in response headers and included in every request log record.
Allowing arbitrary content and lengths makes log correlation harder and lets
one client-controlled header inflate many log records.

## Decision

This record amends [ADR 0012](0012-request-ids-and-one-log-record-per-request.md)
only in its acceptance of client-supplied request IDs without validation.
ADR 0012 remains Accepted: its context propagation, response header, single
summary record and log level rules continue to apply.

- Reuse the first incoming `X-Request-Id` value only when it contains 1–128
  ASCII characters drawn from letters, digits, `.`, `_` and `-`.
- Generate a random base32 ID when the value is missing, empty or invalid.
  Do not trim, truncate or otherwise normalize client values.
- Use the selected ID in the response header and request logging context.
  Leave the incoming request header unchanged.
- Keep request processing unchanged for invalid IDs; do not return an error
  solely because a correlation header is invalid.
- Document the limits and fallback in the API spec and README. Test the
  boundaries, allowed characters, invalid values and response/log consistency.

## Alternatives considered

- **Reject the request.** Makes an optional correlation header prevent useful
  work and requires a new client-visible validation response.
- **Truncate or sanitize the ID.** Changes correlation values and can collapse
  distinct client IDs into the same value.
- **Always generate an ID.** Loses correlation with valid client and proxy IDs.

## Consequences

- Client-controlled IDs have bounded size and a predictable character set in
  logs and responses. Common UUIDs and proxy correlation IDs remain usable.
- Clients sending other characters or longer IDs receive a generated ID and
  must use the response header for subsequent correlation.
- Existing clients with valid IDs keep their current behavior.

# 0023. Liveness and readiness probes

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, operations, observability

## Context

Container platforms need to distinguish a process that cannot answer HTTP from
an instance that should temporarily stop receiving traffic. Restarting a server
does not repair a database outage. During shutdown, routing changes also need
time to propagate before the server stops accepting requests. Frequent probes
would overwhelm normal request logs.

## Decision

This record amends [ADR 0012](0012-request-ids-and-one-log-record-per-request.md)
only in the log level of quiet probe paths. ADR 0012 remains Accepted: its
request IDs, context propagation, response header and single summary record
continue to apply. All other paths retain its status-based log level rules.

- `GET /livez` and HEAD check no dependencies, including the database. They
  return 200 whenever the process can answer HTTP, including while draining.
- `GET /readyz` and HEAD ping the database with a named one-second timeout.
  They return 200 only when the ping succeeds and shutdown has not started;
  otherwise they return 503.
- Both probes live at the root, outside `/api`, and are reserved from the
  client app's catch-all route. Unsupported methods receive a plain-text 405
  with `Allow: GET, HEAD`.
- Responses use `{"status":"ok"}` or `{"status":"unavailable"}` rather than
  the API error shape. HEAD has no body. Every response carries
  `X-Request-Id` and `Cache-Control: no-store`; internal details stay in logs.
- On SIGINT or SIGTERM, mark the instance unready, wait
  `APP_SHUTDOWN_DRAIN_DELAY` (default `0s`), then shut down gracefully with
  the existing timeout. A second signal terminates immediately during either
  phase. Draining is irreversible for that process.
- The request logger accepts an explicit set of quiet paths. Summaries for
  `/livez` and `/readyz` use debug at every status. Readiness logs transitions
  once: warn with the reason on becoming unavailable, info on recovery, and
  info when draining starts. Repeated failures produce no repeated warning.
- Document the GET and HEAD contracts in OpenAPI and verify them through the
  shared test harness, including readiness against a real database.

## Alternatives considered

- **Ping the database for liveness.** Causes restarts during dependency outages
  without repairing the outage.
- **Stop HTTP immediately.** Can leave load balancers routing requests to an
  instance that has already stopped accepting connections.
- **Log each failure at warn or error.** Repeated probes obscure state changes
  and create noise during a prolonged outage.

## Consequences

- Platforms can restart unresponsive processes and remove unavailable
  instances from traffic independently.
- Deployments choose a drain delay appropriate for their routing propagation;
  local Ctrl+C has no additional delay by default.
- Readiness adds one bounded database ping per probe. Probe bodies reveal no
  dependency details, and normal logs emphasize availability transitions.

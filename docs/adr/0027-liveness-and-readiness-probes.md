# 0027. Liveness and readiness probes

- Status: Accepted
- Date: 2026-11-06
- Tags: back-end, operations, observability
- Consolidates: [0023](archive/0023-liveness-and-readiness-probes.md), [0024](archive/0024-json-errors-for-unsupported-probe-methods.md)

## Context

Container platforms need to distinguish a process that cannot answer HTTP from
an instance that should temporarily stop receiving traffic. Restarting a server
does not repair a database outage. During shutdown, routing changes also need
time to propagate before the server stops accepting requests. Frequent probes
would overwhelm normal request logs.

## Decision

- **Liveness** (`/livez`) checks no dependencies, including the database. It
  reports healthy whenever the process can answer HTTP, including while
  draining.
- **Readiness** (`/readyz`) pings the database with a short, bounded timeout.
  It reports healthy only when the ping succeeds and shutdown has not started;
  otherwise it reports unavailable.
- Both probes answer GET and HEAD at the root, outside `/api`, and are
  reserved from the client app's catch-all route. They require no
  authentication.
- **Response contracts.** Probe status responses use a small status body
  (`ok` or `unavailable`) rather than the error shape, and HEAD sends no body.
  Every probe response, errors included, carries the request ID and disables
  caching, and none reveals internal details; those stay in logs. Other
  methods get a method error with an `Allow` header, in the shared JSON error
  shape ([0025](0025-error-responses.md)), written by the central error
  handler. The OpenAPI spec and the README define the exact statuses, bodies,
  headers and timeout.
- **Draining.** On the first SIGINT or SIGTERM the instance becomes unready,
  keeps serving for a configurable drain delay (none by default), then shuts
  down gracefully within the shutdown timeout. A second signal terminates
  immediately during either phase. Draining is irreversible for that process.
- **Logging.** The probe paths are quiet paths: their request summaries log at
  debug at every status ([0026](0026-request-ids-and-request-logging.md)).
  Readiness logs each transition once: a warning with the reason on becoming
  unavailable, info on recovery, and info when draining starts. Repeated
  failures produce no repeated warning.
- The probe contracts are documented in the OpenAPI spec and verified through
  the shared test harness, including readiness against a real database.

## Alternatives considered

- **Ping the database for liveness.** Causes restarts during dependency outages
  without repairing the outage.
- **Stop HTTP immediately.** Can leave load balancers routing requests to an
  instance that has already stopped accepting connections.
- **Log each failure at warn or error.** Repeated probes obscure state changes
  and create noise during a prolonged outage.
- **Plain-text method errors.** Keeps the probes self-contained, but gives
  their clients a second error format.
- **Write the JSON error inside the probe handlers.** Duplicates the central
  handler's response encoding and error logging.

## Consequences

- Platforms can restart unresponsive processes and remove unavailable
  instances from traffic independently.
- Deployments choose a drain delay appropriate for their routing propagation;
  local Ctrl+C has no additional delay by default.
- Readiness adds one bounded database ping per probe.
- Probe bodies reveal no dependency details, and normal logs show availability
  transitions rather than individual probe requests.
- Probe clients handle the same JSON error shape as API clients; a client
  expecting plain-text method errors must handle JSON instead.
- The OpenAPI path descriptions, the response validator in the shared test
  harness and router tests pin the probe bodies, error body and headers.

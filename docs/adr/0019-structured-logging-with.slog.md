# 0019. Structured logging with slog

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, observability

## Context

Logs are the main way to understand what the server did. Free-form text logs
can't be filtered or aggregated reliably, and inconsistent keys or levels
make every query a special case. The standard library now includes a
structured logger.

## Decision

- **All logging goes through the standard library's `log/slog`.** Printing
  with `fmt` and the older `log` package is forbidden by the linter.
- **Logs are written to standard output**, as JSON by default. A text format
  is available for reading locally. The level is configurable.
- **Records are structured:** a short message plus key-value attributes, with
  snake_case keys and one argument style per call, enforced by the linter.
- **Context-aware calls are used whenever a context is available.**
  Request-scoped attributes, such as the request ID, are attached to the
  context once and added to every record automatically.
- **Levels have fixed meanings:**
  - error: an unexpected failure that needs attention;
  - warning: a degraded situation, or a client error in a request summary;
  - info: lifecycle events and request summaries;
  - debug: expected client errors and diagnostic detail.
- **Secrets are never logged.** Error causes are logged in full, but never
  returned to clients.

## Alternatives considered

- **A third-party logger** (zap, zerolog). Faster in benchmarks, but a
  dependency, and `slog` is fast enough for this server.
- **Unstructured text logs.** Readable by eye, but not reliably searchable or
  aggregatable.
- **Writing log files from the application.** File rotation and shipping are
  the platform's job; the process writes to standard output.
- **Passing request-scoped attributes by hand.** Every call site would have to
  remember them, and they would be missed.

## Consequences

- Logs can be filtered and aggregated with standard JSON tools during
  development and by any log platform later.
- Consistent keys and levels make queries reusable across endpoints.
- A small custom handler wraps the standard ones to add context attributes;
  it is the only logging code we own.
- Moving to distributed tracing later adds trace attributes through the same
  context mechanism.
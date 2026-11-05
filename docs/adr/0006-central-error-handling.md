# 0006. Handlers return errors to a central error handler

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, api

## Context

The standard `http.Handler` has no error return, so each handler must write
its own error responses and logs. That spreads status-code decisions across
every endpoint, makes the response shape inconsistent, and makes it easy to
leak internal details or to forget to log.

## Decision

Handlers return `error`, and a single central handler turns errors into
responses:

- **A typed status error** carries the HTTP status, the public response body
  and an optional underlying cause. The cause is logged, never sent.
- **Storage reports domain outcomes as shared sentinel errors** (not found,
  version conflict). The central handler maps them to HTTP. Stores never know
  about HTTP.
- **Anything else is unexpected:** it is logged at error level with its cause,
  and the client gets a generic 500 with no details.
- **Expected client errors** (validation, not found, preconditions) are logged
  at debug level.
- **Edge cases are handled in one place:** HEAD responses get no body, a
  response that has already started is never written to twice, and panics are
  recovered and reported as 500.
- **Unknown API paths and methods** go through the same handler, so they get
  the same error shape, and 405 keeps its `Allow` header.

## Alternatives considered

- **Each handler writes its own errors.** No extra abstraction, but status
  codes, shapes and logging drift between endpoints.
- **A framework's error model.** Ties every handler to the framework's
  context and error types (see the standard-library-first decision).
- **Panic-based control flow** for errors. Hides control flow and makes errors
  hard to test.

## Consequences

- Handlers read as straight-line code ending in `return err`.
- Status codes and response bodies for shared failures are defined once.
- Internal details can't reach clients through an error response.
- Adding a new storage outcome means adding a sentinel and one mapping in the
  central handler.
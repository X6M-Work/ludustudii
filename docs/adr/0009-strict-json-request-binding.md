# 0009. Strict JSON request binding

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, api

## Context

A lenient decoder silently ignores what it doesn't understand: a misspelled
field, a second JSON value appended to the body, a body in another format.
The request then "succeeds" with data the client didn't mean to send. Request
size must also be bounded before the server reads it into memory.

## Decision

Request bodies are bound strictly:

- **JSON only.** A non-empty body must declare the JSON media type, otherwise
  415. Media type parameters such as `charset` are ignored.
- **Unknown fields are rejected** with a binding error (400), at any depth.
- **Exactly one JSON value.** Anything after it, other than white space, is a
  binding error.
- **An empty body of known length binds nothing.** The target keeps its zero
  values, and validation reports missing required fields. An empty body of
  unknown length (chunked) is rejected.
- **Bodies are size-limited.** A known oversized length is rejected before the
  handler runs; bodies of unknown length are cut off while reading. Both
  return 413.
- Binding reads only the body, never query or path values.

## Alternatives considered

- **Lenient decoding** (ignore unknown fields, accept trailing data). More
  forgiving to clients, but typos like `"titel"` are accepted silently and
  the field is lost.
- **Accept form or multipart bodies too.** Not needed by any client; each
  extra format is more parsing surface.
- **No size limit, or a limit only for known lengths.** Leaves the server
  open to memory exhaustion through chunked requests.

## Consequences

- Client mistakes surface as immediate 4xx errors instead of silent data loss.
- Adding a field to a request is backward compatible for the server, but a
  client must not send a field before the server knows it.
- Every endpoint gets the same behavior from one binding function, with no
  per-handler configuration.
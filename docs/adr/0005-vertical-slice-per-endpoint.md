# 0005. Vertical slice per endpoint

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, architecture

## Context

A layered structure (controllers, services, repositories per resource) groups
code by technical role. A single change then touches several packages, and
the shared repository and service types grow with every endpoint. Agents
extend a codebase by copying what is nearest, so the unit of copying should
be small, complete and independent.

## Decision

Each endpoint is its own package inside its service, containing everything it
needs:

- a **handler** that runs a fixed pipeline: path parameters, preconditions,
  binding, sanitizing, validation, storage, response. Cheap checks run before
  expensive ones and before touching the database.
- a **store** with only the queries this endpoint uses, each in its own SQL
  file.
- request and response types private to the package.

Rules and limits shared by a service's endpoints live in one domain package
for that service. There is no separate service or use-case layer between
handler and store. Orchestration goes in the handler until a slice genuinely
needs more.

Each pipeline step the handler uses is a replaceable function field or a
small interface defined next to the handler, so every step can be
unit-tested in isolation. One slice is kept as the reference that new slices
copy.

## Alternatives considered

- **Layered by technical role** (controllers, services, repositories). A
  well-known shape, but changes spread across layers, and repositories become
  large shared types that every endpoint depends on.
- **One package per resource** holding all its handlers and queries. Fewer
  packages, but each endpoint's code is mixed with the others' and harder to
  read, test or copy alone.

## Consequences

- An endpoint can be read, tested and changed without opening any other
  endpoint.
- Some code is repeated across slices (request types, sanitizing), on purpose.
  Shared rules still go in the domain package, so the repetition is only
  wiring.
- Adding an endpoint is mechanical: copy the reference slice, write the spec,
  the SQL and the tests.
- The shape fits CQRS later: read and write slices are already separate.
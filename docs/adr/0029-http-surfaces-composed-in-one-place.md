# 0029. HTTP surfaces composed in one place

- Status: Proposed
- Date: 2026-11-08
- Tags: back-end, architecture

## Context

The binary serves four things over HTTP: probes, the JSON API, the optional
API reference and the client app ([0001](0001-single-binary-services-as-packages.md),
[0003](0003-self-contained-server-binary.md),
[0027](0027-liveness-and-readiness-probes.md)). Startup used to build all of
them in one function and route them through one constructor that took every
dependency at once: logger, client app, reference, database and readiness
state. Each part therefore depended on the others' setup. API tests had to
load the client app, and extracting a part, as 0001 plans, would have meant
untangling the startup code first.

## Decision

- **A surface owns its patterns.** Each part of the HTTP server is a surface
  that registers its own patterns on the root mux (`Mount`). It is built only
  from its own dependencies plus the shared error handler: probes from the
  readiness state, the API from the database, the reference and the client app
  from embedded files.
- **The shared front knows no surface.** `internal/app/router` owns the root
  mux, the middleware chain and the central error handler, and mounts whatever
  surfaces it is given. Surfaces declare quiet paths themselves.
- **The lifecycle knows no surface.** `internal/app/server` serves, drains and
  shuts down any handler. It only needs something to mark stopping.
- **One composition root per binary.** `internal/app` is the only package that
  knows every surface. It loads configuration and dependencies, decides which
  surfaces this binary serves, and runs them. Services still register in one
  place, the API surface (`internal/app/api`).
- depguard enforces these boundaries: the front and the lifecycle import no
  surface, and the API surface imports no other surface.

Observable behavior is unchanged: one listener, the same paths, statuses,
headers and logs.

## Alternatives considered

- **Separate listeners now** (for example, probes on an admin port). This adds
  a port to configure and operate before any platform needs it, and changes
  the contract in 0027. A composition root can add a listener later without
  changing the surfaces.
- **Separate binaries now.** This contradicts 0001 and 0003 while the
  boundaries are still moving.
- **Keep one router constructor and split it into files.** Cheaper, but every
  part still depends on every other part's setup, and tests still load the
  client app to exercise the API.

## Consequences

- A future binary, such as an API-only one, is a new composition root that
  mounts a subset of the existing surfaces.
- Tests build only the surfaces they exercise. Service API tests no longer
  load the client app.
- The client app's catch-all must stay method-less while it shares the root
  mux with method-less prefixes such as `/api/`. This is the one coupling
  between surfaces that remains.
- Readiness still pings the database. A binary without a database needs a
  readiness state that has no dependencies, which will be a change to
  `health.State` when that binary exists.

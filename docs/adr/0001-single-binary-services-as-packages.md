# 0001. Single binary now, services as packages

- Status: Proposed
- Date: 2026-11-04

## Context

Ludus Studii is meant to become a set of microservices. Today the product is
small, and its domain boundaries are still being discovered. Splitting into
separate deployables now would add a build, deploy, network and data layer per
service. It would also freeze boundaries before we know where they really lie;
a wrong boundary is far more expensive to move across a network than across a
package.

## Decision

Ship one binary (`cmd/server/api`) that serves the API, the client app and the
API reference. Structure it so each service can be extracted later without a
rewrite:

- **A service is a bounded context, not a resource.** It is a package under
  `internal/services/<name>`. A new resource that is tightly coupled to an
  existing service's data, and changes with it, becomes new slices in that
  service rather than a new service.
- **Services do not import each other.** They interact only through a narrow
  interface wired in `internal/app` or, once it exists, through events. Shared
  code that isn't domain logic belongs in `internal/kit`.
- **Each table is owned by exactly one service**, and only that service's
  stores read or write it.
- **No cross-service joins or foreign keys.** One service refers to another's
  rows by ID only. Consistency across services comes from events (for example,
  "card deleted"), not from the database.
- **Infrastructure stays out of services.** Services depend on `kit`, never on
  `infra` or `app` (already enforced by depguard).

## Extraction plan

Splitting is planned, not hypothetical. A service is extracted when it has a
concrete reason to run separately: a different runtime profile, independent
scaling, or failure isolation. The first likely candidate is AI evaluation and
question generation: slow, external, retry-heavy work that shouldn't share a
process with request handling. Each extraction gets its own ADR.

## Alternatives considered

- **Microservices from the start.** Rejected for now: high operational cost
  (deploys, service discovery, a broker, distributed tracing, cross-service
  testing) while the boundaries are still unstable.
- **A plain monolith with shared tables and free imports.** The fastest to
  write, but a later split would need untangling joins, foreign keys and
  imports across the whole codebase.

## Consequences

- One build, one deploy, one database, and simple local development and
  integration testing.
- Extraction becomes mostly mechanical: move the package, give it its own
  `main` and database, and replace in-process calls with network calls or
  events.
- Some things are harder now, on purpose. No foreign keys across services
  means referential integrity between them is enforced by application code and
  events, not by Postgres.
- Migrations live in one directory for all services. An extraction must split
  out the owning service's tables and migrations.
- The no-import rule must be enforced by `make check` (a depguard rule or a
  small architecture test) once a second service exists. Until then, this ADR
  is the rule.
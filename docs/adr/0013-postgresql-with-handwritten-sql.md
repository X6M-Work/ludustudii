# 0013. PostgreSQL with hand-written SQL

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, database

## Context

The application needs a relational database with strong constraints,
transactions and database-generated, time-ordered IDs. How Go code talks to it decides how visible and reviewable
the queries are, and how much of the database's power the code can use.

## Decision

- **PostgreSQL is the only database.** Its specific features are used
  freely: `CHECK` constraints, CTEs, `RETURNING`, `timestamptz` and
  database-generated IDs. Portability to other databases is not a goal.
- **Access goes through the standard `database/sql` interface**, backed by a
  pure-Go PostgreSQL driver.
- **SQL is written by hand.** No ORM, no query builder and no generated query
  code. Every query the application runs is in the source as plain SQL.
- **Data rules live in the database as well as in code.** Limits and
  invariants that validation enforces are mirrored as constraints, so
  invalid data can't be written by any path.

## Alternatives considered

- **An ORM** (GORM, ent). Less boilerplate for simple CRUD, but queries are
  hidden behind method chains, and conditional writes, CTEs and
  Postgres-specific features end up as raw SQL anyway.
- **A query builder** (squirrel, goqu). Composable, but SQL is harder to read
  than the SQL itself, and it adds a dependency.
- **Generated query code from SQL** (sqlc). Typed and safe, but adds a code
  generator to the build and its conventions to every store, against the
  standard-library-first policy.
- **The driver's native interface instead of `database/sql`.** Faster and with
  more features (batching, `COPY`, notifications), but other libraries (the
  migration tool, the test harness) expect `database/sql`. Native features
  remain reachable through the underlying connection when a slice needs them.

## Consequences

- Every query can be read, explained and tuned as written.
- More code per store: scanning rows and mapping errors by hand.
- Mistakes in SQL or parameter order show up only when the query runs, so
  stores are covered by integration tests against a real database.
- Moving to another database would mean rewriting queries; that is accepted.
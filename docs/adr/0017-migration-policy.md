# 0017. Migration policy

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, database

## Context

The schema changes with almost every feature. Changes need to be repeatable
across development, test and future production databases, reversible while a
feature is being built, and controlled at deploy time. While a feature is in
progress, it often produces several small migrations that are clearer as one.

## Decision

- **Migrations are versioned SQL files**, each with an `Up` and a `Down`.
  They are applied in order by a migration tool.
- **Every `Down` must restore the schema exactly as it was before its `Up`.**
  A test applies and rolls back each migration in turn and compares full
  schema snapshots, and checks that rolling back everything leaves an empty
  schema.
- **The server never migrates on startup.** Migrations run as an explicit
  step, before the new server version starts.
- **Test databases are built from the same migrations**, so tests always run
  against the real schema.
- **Squashing is permitted per feature.** The migrations belonging to one
  feature may be combined into one, and existing migrations may be edited,
  but only after asking. Agents never squash or edit an existing migration on
  their own; they add a new one.

## Alternatives considered

- **Migrate on server startup.** One step fewer to deploy, but several
  instances starting together race for the migration, the server needs
  schema-changing privileges, and a failed migration becomes a failed start.
- **Strictly append-only history.** Safest once real data exists, but it
  keeps every intermediate step of unfinished features forever.
- **Declarative schema diffing** (describe the target schema, let a tool
  compute the change). Less to write, but the generated changes are harder to
  review, and it adds a tool.
- **ORM auto-migration.** Ruled out by the decision to write SQL by hand.

## Consequences

- Deploys have two steps: migrate, then start the new server. The schema must
  stay compatible with the server version that is still running during that
  window.
- Squashing or editing a migration rewrites history: every database that
  applied the old version must be recreated. That's cheap while no
  environment holds data that must be kept, and this decision must be
  revisited before that changes.
- Writing and maintaining `Down` migrations costs effort, and the round-trip
  test enforces it.
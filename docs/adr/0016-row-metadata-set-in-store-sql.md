# 0016. Row metadata set in store SQL

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, database

## Context

Every table needs to know when a row was created and last changed, and
user-editable tables also need a version for optimistic concurrency.
PostgreSQL has no `ON UPDATE` column clause, so `updated_at` must be set
either by a trigger or by every `UPDATE`. The project prefers explicit,
visible code, and most new stores are written by copying existing ones.

## Decision

- **Every table has `created_at` and `updated_at`**, both `timestamptz NOT NULL`
  with **no default**, and a constraint that `updated_at >= created_at`.
- **Stores set them explicitly in SQL:**
  - `INSERT` sets both to `now()`;
  - every `UPDATE` sets `updated_at = now()`, in the same statement that bumps
    `version` where the table has one;
  - nothing writes `created_at` after insert.
- **Timestamps come from the database clock** (`now()`), never from the
  application clock.
- **User-editable tables have a `version`** that starts at 1 and is
  incremented by every update, as required by optimistic concurrency.
- **Timestamps are internal.** They aren't returned by the API until a
  feature needs them.

## Alternatives considered

- **A `BEFORE UPDATE` trigger.** Impossible to forget, but invisible when
  reading the store, and it fires on bulk data fixes too.
- **Column defaults** (`DEFAULT now()`). Less to write, but an `INSERT` that
  forgets the columns silently succeeds.
- **Application clock** (`time.Now()` in Go). Easy to fake in tests, but every
  instance has its own clock, and values differ from what the database
  considers "now".
- **Using `updated_at` as the version.** Clock precision and skew make it
  unreliable for detecting conflicts.

## Consequences

- Every write to metadata is visible in the store's SQL.
- An `INSERT` that forgets the timestamps fails on `NOT NULL`.
- An `UPDATE` that forgets `updated_at` is **not** caught by the database;
  only review, tests and the reference slice catch it.
- `now()` is the transaction start time, so all rows written in one
  transaction share the same timestamp.
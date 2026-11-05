# 0010. Optimistic concurrency with ETag and If-Match

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, front-end, api

## Context

A user can edit the same resource from two tabs or devices. Without a check,
the second write silently overwrites the first ("lost update"). Locking rows
while a user edits is not possible over HTTP, and last-write-wins loses data.

## Decision

Every user-editable resource carries an integer `version`, starting at 1.

- **Every successful update increments the version** in the same statement
  that changes the data.
- **The version is exposed two ways**, both derived from the same value: a
  strong `ETag` header (a quoted decimal, such as `"3"`) on single-resource
  responses, and a `version` field in response bodies, including list items,
  which have no per-item header.
- **Updates and deletes require `If-Match`** with exactly one strong ETag. The
  write applies only if the stored version still matches, checked in the
  write statement itself, in one round trip.
- **Responses:**
  - missing or malformed `If-Match` → 428 (this includes `*`, lists and weak
    tags);
  - unknown resource → 404, which takes precedence over a version mismatch;
  - stale version → 412;
  - a successful update returns the new ETag.
- After a 412, the client reloads; the server never merges.

## Alternatives considered

- **Last write wins.** Simple, but loses data without telling anyone.
- **Version in the request body, 409 on conflict.** Works, but reinvents what
  HTTP already standardizes.
- **`If-Unmodified-Since` with timestamps.** One-second precision is too
  coarse, and clocks can disagree.
- **Database-internal row versions** (such as Postgres `xmin`). No extra
  column, but internal, wrapping and meaningless to clients.
- **Pessimistic locking.** Holding locks across user think time doesn't work
  over stateless HTTP.

## Consequences

- No silent lost updates. Conflicts surface as 412, and the client decides
  what to show.
- Clients must keep the version they read and send it back; the client app
  does this for every edit and delete.
- The rule applies to every new editable table, and nothing enforces it but
  review and the reference slice.
- The version doubles as an ordering and de-duplication key if events are
  published later.
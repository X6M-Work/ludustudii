# 0011. Canonical IDs and keyset pagination

- Status: Accepted, Amended by [0022](0022-opaque-pagination-cursor-contract.md)
- Date: 2026-11-05
- Tags: back-end, api

## Context

Resource IDs are UUIDv7: time-ordered, so sorting by ID sorts by creation
time. Clients and humans may type or copy IDs in either letter case. Lists
grow without bound and need pagination that stays fast and doesn't skip or
repeat items while data changes.

## Decision

- **IDs in responses are always canonical:** lowercase, hyphenated UUIDv7.
- **IDs in requests are accepted in any letter case** and lowercased before
  validation. Anything other than a valid UUIDv7 after that is a validation
  error.
- **Lists use keyset pagination** on the ID, newest first. A page is "items
  with an ID below the cursor", with a bounded page size and a default.
- **The cursor is the ID of the last item** of the previous page, returned as
  `next_cursor`, and `null` on the last page. It follows the same ID rules.

## Alternatives considered

- **Offset pagination** (`page`, `offset`). Familiar, but slow on deep pages,
  and items shift between pages when rows are added or deleted.
- **An opaque, encoded cursor.** Lets the sort key change without breaking
  clients, but adds encoding and decoding for no current need.
- **Reject non-canonical ID case.** Strict, but punishes clients for a
  difference with no meaning.

## Consequences

- Page queries use the primary key index and stay fast at any depth.
- No skipped or duplicated items when new rows are inserted between requests.
- The cursor's format is part of the API contract. Adding another sort order
  or filters later means introducing a different cursor format, which is a
  breaking change unless clients already treat it as opaque.
- Jumping to an arbitrary page number isn't possible, only next-page
  navigation.
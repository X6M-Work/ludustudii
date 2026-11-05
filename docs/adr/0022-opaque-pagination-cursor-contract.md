# 0022. Opaque pagination cursor contract

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, api

## Context

ADR 0011 describes the cursor as the last item's ID and exposes its format
as part of the API contract. Describing tokens as opaque while retaining
UUID constraints in the schema still binds generated clients and validators
to that implementation. A UUID-specific cursor error code does the same.

## Decision

This record amends [ADR 0011](0011-canonical-ids-and-keyset-pagination.md)
only in its public cursor contract. Its resource ID rules, keyset pagination,
newest-first order and page size rules continue to apply.

- **Cursors are opaque strings in the API contract.** Both the `cursor`
  parameter and `next_cursor` schema use `type: string` without a UUID format
  or pattern. Clients pass returned tokens unchanged and must not parse or
  construct them. `next_cursor` remains nullable on the final page; an omitted
  or empty `cursor` starts the first page.
- **Invalid tokens report `validation_error` with `cursor/invalid_cursor`.**
  Path IDs continue to report `id/invalid_uuid7`.
- **The last item's canonical UUIDv7 ID is the current implementation.**
  The server continues to validate that form internally and accept cursors
  in any letter case. This representation is not a public format guarantee.

## Alternatives considered

- **Keep UUID constraints and error codes.** Exposes the implementation even
  if the description calls the cursor opaque.
- **Encode tokens now.** Adds encoding and decoding without a current need;
  opacity is a client contract and does not require encoding.

## Consequences

- Future token representations do not require changing a UUID-constrained
  public schema or error code. Existing paging behavior stays the same.
- Generated clients treat tokens as strings. Clients that inspect UUID
  contents or construct tokens depend on an implementation detail.
- Spec, API, domain and sanitizer contract tests cover opaque schemas,
  cursor-specific errors and the current internal validation.

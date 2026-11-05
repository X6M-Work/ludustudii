# 0014. SQL in embedded query files

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, database

## Context

Hand-written SQL kept in Go string literals gets no editor support: no
syntax highlighting, formatting or SQL linting. Long queries become hard to
read, and building SQL from string fragments at runtime invites both bugs and
injection.

## Decision

- **Each query lives in its own `.sql` file** next to the store that runs it,
  and is embedded into the binary with `//go:embed`. Stores contain no inline
  SQL strings.
- **Each file starts with a header comment** listing its positional
  parameters in order (`-- $1 title, $2 description`).
- **Result columns are named** with `AS` when they aren't plain column
  references, so the query documents what each scanned value means.
- **No SQL is assembled at runtime.** When a query has variants (such as the
  first page of a list and the pages after a cursor), each variant is its own
  file.
- **A conditional write is one statement.** The write and the checks that
  explain its outcome (did it apply, does the row exist) run together in a
  single statement using CTEs, and return flags the store maps to domain
  errors. One round trip, and no race between the write and the check.

## Alternatives considered

- **Inline string constants.** Everything in one file, but no SQL tooling,
  and long queries clutter the Go code.
- **Building queries from fragments at runtime.** Flexible for filters, but
  harder to read, and each combination is a query nobody has seen whole.
- **A write followed by a separate existence query.** Simpler SQL, but two
  round trips, and the row can change between them.

## Consequences

- SQL gets full editor support and can be run by hand against a database
  while developing.
- Parameter order is documented only by the header comment; a mismatch is
  caught by integration tests, not by the compiler.
- Variants multiply files. Once filters or sort options arrive, the number of
  combinations may force a decision on safe dynamic SQL, recorded separately.
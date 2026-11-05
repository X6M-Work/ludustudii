# 0008. Canonicalize text input before validating

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, api

## Context

The same text can arrive in different byte forms: precomposed or decomposed
Unicode characters, and stray leading or trailing white space. Stored as-is,
these break equality, uniqueness and length checks, and make two identical-
looking values differ. Users shouldn't be rejected for an accidental trailing
space.

## Decision

- **Sanitize, then validate.** Input text is normalized to Unicode NFC and
  trimmed before validation runs.
- **Validators reject non-canonical text.** A validator accepts only text
  already in canonical form, so the validators alone define what may be
  stored, and skipping the sanitizer can't let bad data in.
- **Fuzz tests prove the two agree:** anything the sanitizer outputs passes
  the format validators.
- **Lengths are counted in Unicode code points**, not bytes.
- **The database enforces the same limits** with `CHECK` constraints, as a
  last line of defense.
- Stored and returned text is always canonical.

## Alternatives considered

- **Reject non-canonical input.** Strict and simple, but punishes users for
  invisible differences such as a trailing space.
- **Store as given, normalize when comparing.** Every query and comparison
  must remember to normalize, and the stored data stays inconsistent.
- **Count bytes.** Penalizes non-Latin scripts (Cyrillic uses two bytes per
  letter in UTF-8).
- **Count grapheme clusters.** Closest to what users see, but needs a
  third-party library, and the database can't enforce the same rule.

## Consequences

- Data is consistent at rest: equal-looking text is equal.
- Limits in the API, the validators and the database agree, because they
  count the same unit.
- Clients may send text in any normalization form; they should still validate
  the canonical form to give accurate length feedback.
- A new text rule needs a sanitizer step, a validator and an extended fuzz
  test.
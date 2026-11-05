# Architecture Decision Records

Each record captures one decision that is hard to reverse, or that a
reasonable person might make differently: the context, the choice, the
alternatives and the trade-offs. The code shows *what* was decided; these
records explain *why*.

## Rules

- Copy `template.md` to `NNNN-short-title.md`, using the next free number.
  Numbers are never reused or renumbered.
- An Accepted record is not edited, except for its status line. To change a
  decision, write a new record that supersedes it, and set the old record's
  status to `Superseded by NNNN`.
- Keep each record to about a page. Anything longer is a design doc.
- Add every record to the index below.

## Index

| No. | Decision | Status | Tags |
| --- | --- | --- | --- |
| [0001](0001-single-binary-services-as-packages.md) | Single binary now, services as packages | Proposed | back-end, architecture |
| [0002](0002-standard-library-first-minimal-dependencies.md) | Standard library first, minimal dependencies | Proposed | back-end, dependencies |
| [0003](0003-self-contained-server-binary.md) | Self-contained server binary | Proposed | back-end, front-end, architecture |
| [0004](0004-spec-first-open-api-contract.md) | Spec-first OpenAPI contract | Proposed | back-end, api |
| [0005](0005-vertical-slice-per-endpoint.md) | Vertical slice per endpoint | Proposed | back-end, architecture |
| [0006](0006-central-error-handling.md) | Handlers return errors to a central error handler | Proposed | back-end, api |
| [0007](0007-one-json-error-shape-with-machine-readable-codes.md) | One JSON error shape with machine-readable codes | Proposed | back-end, front-end, api |
| [0008](0008-canonicalize-text-input-before-validating.md) | Canonicalize text input before validating | Proposed | back-end, api |
| [0009](0009-strict-json-request-binding.md) | Strict JSON request binding | Proposed | back-end, api |
| [0010](0010-optimistic-concurrency-with-etag-and-if-match.md) | Optimistic concurrency with ETag and If-Match | Proposed | back-end, front-end, api |
| [0011](0011-canonical-ids-and-keyset-pagination.md) | Canonical IDs and keyset pagination | Proposed | back-end, api |
| [0012](0012-request-ids-and-one-log-record-per-request.md) | Request IDs and one log record per request | Proposed | back-end, api, observability |

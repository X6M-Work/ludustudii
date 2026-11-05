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

| No. | Decision | Status |
| --- | --- | --- |
| [0001](0001-single-binary-services-as-packages.md) | Single binary now, services as packages | Proposed |
| [0002](0002-standard-library-first-minimal-dependencies.md) | Standard library first, minimal dependencies | Proposed |

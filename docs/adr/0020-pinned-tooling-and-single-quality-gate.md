# 0020. Pinned tooling and a single quality gate

- Status: Accepted
- Date: 2026-11-05
- Tags: back-end, tooling

## Context

Much of the code is written by agents, which can only verify their work
through commands. When "done" depends on remembering several commands, or on
tools whose versions differ between machines, results vary and failures
appear late. Developer tools also come with large dependency trees that
shouldn't mix with the application's dependencies.

## Decision

- **One command defines "done":** `make check` runs formatting, lint, module
  tidiness, generated-file freshness, unit tests and integration tests, in
  that order (fast to slow), and stops at the first failure. People, agents
  and any future CI run the same command.
- **Make is the single entry point** for development workflows, and every
  target is listed with a description in its help output.
- **Tool versions are pinned:**
  - Go developer tools are tracked in a separate Go module and run with
    `go tool`, so their dependencies never enter the application module.
  - The linter is a standalone binary at a pinned version, installed by the
    development container, as its maintainers recommend.
- **The lint configuration is explicit.** Every enabled linter is listed, and
  every deliberately disabled one carries a reason. A one-off exception must
  name the linter and explain itself next to the code.
- **The development environment is reproducible**: a development container
  provides the toolchain and separate development and test databases.
- **The vulnerability scan is a separate target**, not part of the gate. It
  depends on an external vulnerability database, so it can fail for reasons
  unrelated to a change.

## Alternatives considered

- **Tools tracked in the application module.** One module fewer, but tools'
  dependency trees (cloud SDKs, database drivers) end up in the application's
  module graph and can force version upgrades on runtime dependencies.
- **Installing tools at `@latest`.** No version tracking, so each machine
  drifts.
- **Separate scripts for CI.** Local results and CI results drift apart.
- **Another task runner** (Task, just, Mage). Nicer syntax, but one more tool
  to install, while Make is already everywhere.

## Consequences

- An agent can verify its own work with one command and get the same result
  as a person.
- The full gate needs the test database and runs the whole suite, so focused
  test targets exist for iterating.
- Upgrading a tool is a deliberate change to a pinned version.
- Vulnerabilities aren't caught by the gate; the scan has to be run on its
  own until CI runs it on a schedule.
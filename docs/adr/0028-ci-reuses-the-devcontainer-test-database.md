# 0028. CI reuses the devcontainer test database and tool versions

- Status: Accepted
- Date: 2026-11-07
- Tags: back-end, tooling

## Context

[ADR 0020](0020-pinned-tooling-and-single-quality-gate.md) makes `make check`
the single gate and expects CI to run the same command. That only works if CI
also runs it in the same environment: the same tool versions, and a test
database configured like the devcontainer's. Every copy of a version or a
database setting is a place where local and CI results can drift apart.

## Decision

- **CI starts the test database from the devcontainer's compose definition**
  and runs everything else directly on the runner. The compose file is the
  only definition of the test database.
- **CI reads each tool version from the same single source as the
  devcontainer.** `make check` fails when a copy that can't read the source
  directly disagrees with it.
- **Every CI check runs a Make target**, so its rules live in the Makefile
  and can be run locally. `make check` is the gate for code; checks on pull
  request metadata (such as the title format) run their own target, whose
  self-test is part of `make check`. The vulnerability scan runs on a
  schedule, outside the gate (ADR 0020).

## Alternatives considered

- **GitHub service containers.** These can't apply the test server's
  configuration (its flags and tmpfs storage), and they would restate the
  database definition in a second place.
- **Running the whole devcontainer in CI.** This gives the closest parity, but
  every run would also set up tooling the gate doesn't need, which makes CI
  slower and adds a layer to debug.
- **A hosted test database.** This needs secrets and network access, is shared
  between concurrent runs, and isn't defined in the repository.

## Consequences

- A change to the test database or a tool version is made in one place, and CI
  picks it up.
- The test database service must stay startable on its own, without the
  application container.
- `make check` enforces that versions agree. Only this record enforces that CI
  reuses the compose definition.

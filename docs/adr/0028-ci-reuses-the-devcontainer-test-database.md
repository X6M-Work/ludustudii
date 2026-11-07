# 0028. CI reuses the devcontainer test database and tool versions

- Status: Proposed
- Date: 2026-11-07
- Tags: back-end, tooling

## Context

[ADR 0020](0020-pinned-tooling-and-single-quality-gate.md) makes `make check`
the single gate and expects CI to run the same command. That only holds if CI
also runs it against the same environment: the same Go and golangci-lint
versions, and a test database configured like the devcontainer's `db_test`
(PostgreSQL 18 on tmpfs, with `fsync`, `synchronous_commit` and
`full_page_writes` off). Each copy of a version or database setting is a place
where local runs and CI runs can quietly drift apart.

## Decision

- **CI starts the test database from the devcontainer's compose file.** The
  workflow runs `docker compose -f .devcontainer/docker-compose.yml up -d --wait
  db_test` and runs everything else directly on the runner. `db_test` has a
  `pg_isready` healthcheck, so `--wait` returns only once Postgres accepts
  connections.
- **CI reads tool versions from the same sources as the devcontainer:**
  - Go comes from `backend/go.mod`. `make versions-check`, part of
    `make check`, fails when the devcontainer's `golang:` image tag or
    `backend/tools/go.mod` disagrees with it.
  - golangci-lint is pinned once, by `GOLANGCI_LINT_VERSION` in the `Makefile`.
    The devcontainer and CI both install it with `make install-golangci-lint`,
    and `make versions-check` fails when the installed binary differs.
- **CI runs `make check` and nothing else as its gate.** The vulnerability scan
  (`make vuln`) runs in a separate scheduled workflow, because it depends on an
  external database (ADR 0020).

## Alternatives considered

- **GitHub service containers.** These are native to Actions, but they can't
  pass the `postgres -c` flags or the tmpfs mount, so CI would test against a
  differently configured server. They would also restate the image and
  credentials in a second file.
- **Running the whole devcontainer in CI** (for example with the devcontainers
  CLI). This gives the closest parity, but every run builds the app container
  and runs `post-create.sh`, including frontend and apt installs that the gate
  doesn't need. That makes CI slower and adds another layer to debug when it
  fails.
- **A hosted test database.** This needs secrets and network access, is shared
  between concurrent runs, and is configured nowhere in the repository.

## Consequences

- A change to the test database (image, flags, credentials, port) is made once,
  in the compose file, and CI picks it up. The workflow restates only the
  connection string.
- Upgrading Go or golangci-lint is one deliberate change. A partial upgrade
  fails `make check` locally and in CI.
- CI depends on the compose file staying usable on its own: `db_test` must not
  gain dependencies on the `app` service.
- CI does not install frontend dependencies, so the Scalar bundle isn't
  embedded there and the API docs tests take their "no script" path.
- Enforced by `make versions-check` and the workflow itself; the reuse of the
  compose file is enforced only by this record.

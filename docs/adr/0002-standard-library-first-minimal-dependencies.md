# 0002. Standard library first, minimal dependencies

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, dependencies

## Context

Go's standard library covers most of what a backend needs: HTTP serving and
routing with method patterns, JSON, `database/sql`, structured logging with
`slog`, embedding, testing and fuzzing. Every third-party module adds code we
don't control: upgrades, breaking changes, transitive dependencies,
vulnerabilities and conventions that leak into our own code. The project also
values explicit control and a deep understanding of what runs, and code
written by agents copies whatever patterns it finds.

## Decision

Use the standard library by default. Add a third-party module only when all
of these hold:

- The standard library has no reasonable equivalent.
- The problem is hard or risky to get right ourselves.
- Owning the code would cost more than tracking the dependency.

A candidate must also:

- be actively maintained and widely used;
- have a permissive license (MIT, BSD or Apache 2.0);
- be pure Go, since the binary is built with `CGO_ENABLED=0`;
- have a small transitive dependency tree.

Prefer `golang.org/x/...` over other third-party modules.

The bar depends on where the code ends up:

- **Runtime** (linked into the server binary): the strictest bar.
- **Tests and build-time tools** in the backend module: a lower bar, since
  they don't ship.
- **Developer tools** live in a separate Go module and run with `go tool`.
  Their dependency trees never enter the backend module.

Ask before adding any module (also stated in AGENTS.md).

## Alternatives considered

- **A web framework** (Echo, Gin, Chi). `net/http` routing and middleware
  cover the need; a framework's context type and error model would spread
  through every handler.
- **An ORM or query builder** (GORM, sqlc, squirrel). Hand-written SQL keeps
  queries visible and reviewable.
- **Libraries for validation, configuration and logging** (go-playground/validator,
  viper, zap/zerolog). Small in-house packages built on the standard library
  fit the project's rules exactly.
- **Container-based test tooling** (testcontainers, dockertest). The test
  harness clones a migrated template database on a dedicated test server
  instead.

## Consequences

- A small, auditable runtime dependency set. Upgrades and `govulncheck`
  results stay easy to review.
- We own more code (HTTP helpers, validation, the test harness, logging), so
  it needs the same test coverage as feature code.
- No framework conventions to lean on. The reference slice, AGENTS.md and
  these records carry the conventions instead.
- Developer tools can pull in large trees (the goose CLI brings database
  drivers and cloud SDKs) without touching the backend module.
- Enforced by review and the AGENTS.md rule, not by tooling. Any change to
  the direct requirements in `go.mod` is visible in the diff.
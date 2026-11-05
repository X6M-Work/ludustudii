# 0018. Configuration from environment variables

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, configuration

## Context

The server runs locally, in tests and later in deployed environments, each
with different settings and secrets. Configuration read from many places, or
at unpredictable times, makes behavior hard to reason about. A missing or
invalid value then fails deep inside a request instead of at startup.

## Decision

- **Environment variables are the only source of configuration.** No config
  files and no command-line flags.
- **They are read in exactly one place, once, at startup**, into a typed
  configuration value. Everything else receives its settings as parameters.
  Reading the environment anywhere else is forbidden by the linter; the only
  exception is the test harness, for its test database connection.
- **Every value is parsed and validated at load**: types, ranges, formats.
  Required values must be set; optional ones fall back to documented
  defaults.
- **All problems are reported together**, and the server refuses to start
  while any remain.
- **The effective configuration is logged at startup**, with secrets
  redacted.
- Every variable is documented in the README with its default.

## Alternatives considered

- **Configuration files** (YAML, TOML). Good for large, nested settings, but
  files must be shipped next to the binary, against the self-contained binary
  decision.
- **Command-line flags.** Explicit, but awkward for secrets, and they don't fit
  container platforms as naturally as environment variables.
- **A configuration library** (viper, envconfig). Saves some parsing code, but
  adds a dependency for something the standard library already handles.
- **Reading variables where they are needed.** No central list, hidden
  defaults, and failures happen at request time.
- **Stopping at the first invalid value.** Simpler, but fixing a
  misconfigured environment then takes one restart per mistake.

## Consequences

- A misconfigured deployment fails immediately, with a complete list of what
  to fix.
- Tests build configuration values directly and never touch the process
  environment.
- Adding a setting means changing one loader and the README table.
- Secrets live in the process environment. That's adequate now; a secret
  manager can feed the same variables later.
# 0003. Self-contained server binary

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, front-end, architecture

## Context

The server delivers three things: the JSON API, the single-page client app
and, outside production, an interactive API reference. Each could be deployed
on its own, but every separate artifact adds a deploy step, a way for the
client and API versions to drift apart, and files that must sit at the right
path on the host. The API reference also needs a large third-party rendering
script that production users don't need.

## Decision

Build one statically linked binary (`CGO_ENABLED=0`) that needs nothing at
runtime except its environment configuration:

- **The client app is embedded** with `//go:embed` and served by the same
  binary as the API, from the same origin.
- **The API reference page and the bundled OpenAPI spec are always embedded.**
  They are small, and serving the spec as-is needs no parsing at startup.
- **The reference's rendering script is opt-in, behind a build tag.** Builds
  without the tag leave it out entirely. The script is self-hosted; the
  reference makes no external requests.
- **Misconfiguration fails at startup, not at request time:**
  - Enabling the reference on a binary built without the tag is a deployment
    mistake, and the server refuses to start.
  - A tagged build whose script is missing (a checkout without frontend
    dependencies) starts with a warning and without the reference, so local
    development isn't blocked.
- **Release builds require a real client build.** Development builds embed a
  placeholder page so the backend compiles without the frontend, and the
  release build refuses to run while the placeholder is present.

Schema migrations are not part of this decision. The server doesn't apply
them; that's covered by the migration policy.

## Alternatives considered

- **Serve the client app separately** (static hosting or a CDN, API on another
  origin). It means two deploys and versions that can drift, and adds CORS
  and cookie configuration. A CDN can still be put in front of this binary
  later.
- **Read assets from disk at runtime.** Every deployment would have to ship
  files next to the binary at the configured path, and a mismatch between
  binary and files shows up only at runtime.
- **Always embed the reference.** A larger production binary, and one setting
  away from exposing the docs UI in production.
- **Load the rendering script from a public CDN.** No embedding cost, but it
  makes external requests and depends on a third party being available.

## Consequences

- One artifact is one version of the client, the API and its contract.
  Deploys and rollbacks are atomic.
- The client and the API are released together: a client-only change still
  means rebuilding and redeploying the binary.
- Build order matters (client first, then backend). The build enforces this
  instead of relying on people to remember.
- Client assets are content-hashed and served with immutable cache headers,
  so putting a CDN in front later needs no change to the binary.
- The binary grows with the client bundle. Production builds stay smaller by
  leaving out the reference script.
# 0004. Spec-first OpenAPI contract

- Status: Proposed
- Date: 2026-11-05
- Tags: back-end, api

## Context

The API is consumed by our own client app now and possibly by other clients
later. Without one authoritative description, the server, the client and the
documentation drift apart, and every disagreement is found by a user. Agents
writing new endpoints also need a contract to work against.

## Decision

The OpenAPI document is the source of truth for the HTTP API.

- **Spec first.** A change to the API starts with a change to the spec, in
  the same change as the code and tests.
- **The spec is split across files** for editing (paths, components) and
  bundled into a single document. The bundle is committed and served by the
  API reference, and a test fails when it is stale.
- **Every response in API tests is checked against the spec**: the status
  must be documented, and headers and body must match its schema. A status
  documented without content must have an empty body.
- **Requests are not validated against the spec**, at runtime or in tests.
  The server does its own input validation with its own error shape, and
  tests must be able to send invalid input.
- **Server code is written by hand**, not generated from the spec. The
  response checks keep the two aligned.

## Alternatives considered

- **Code-first** (generate the spec from annotations or code). The spec
  becomes a by-product that is easy to leave incomplete, and the contract
  changes whenever the code does.
- **Generated server stubs** from the spec. Adds a code generator and its
  runtime conventions to every handler, against the standard-library-first
  policy, and generated code is harder for agents to follow than a
  hand-written reference slice.
- **Request validation middleware driven by the spec.** It would duplicate the
  server's own validation, impose the library's error format, and add a
  runtime dependency.

## Consequences

- Documentation, client expectations and server behavior can't silently
  diverge: an undocumented status, header or field fails a test.
- Every API change touches at least three places (spec, code, tests). This is
  intended.
- Response validation is only as good as test coverage; an untested path can
  still drift.
- Requests that the spec would reject but the server accepts are not caught
  automatically; the server's validation and its tests cover that side.
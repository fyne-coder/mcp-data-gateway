# Changelog

All notable changes to this project will be documented in this file.

This project follows semantic versioning.

## [0.1.0] - 2026-06-02

Initial pre-alpha source release.

### Implemented

- OIDC/JWKS bearer validation on `POST /mcp` via `github.com/coreos/go-oidc/v3`; `/healthz` and `/readyz` stay unauthenticated.
- Payload-safe, one-event-per-request audit for `/mcp` auth and tool decisions (no raw tokens, bodies, tool args, or row payloads).
- Deterministic `doctor` command: config and audit sink checks by default; opt-in JWKS reachability, allowed/denied token file checks, and optional live MCP tool invocation checks.
- SDK-backed MCP Streamable HTTP on `/mcp` via `github.com/modelcontextprotocol/go-sdk` (`initialize`, `tools/list`, `tools/call`).
- Public-safe remote MCP conformance fixtures for `initialize`, `tools/list`, allowlisted `tools/call`, and forbidden-table error checks.
- Read-only `postgres_select` MCP tool: structured allowlisted inputs, `github.com/jackc/pgx/v5` execution, group tool-pack policy, table/column allowlists, passthrough result shaping only.

### Scaffold

- Public project scaffold, CLI (`version`, `doctor`, `serve`), configuration contract, and example config.
- Architecture, requirements, threat model, audit schema, roadmap, development, deployment, operations, and publishing docs.
- CI workflow and local `make ci` gate.

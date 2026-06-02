# Roadmap

Status: **0.1.0 pre-alpha source release**. The sections below separate what is in-tree today from what is still planned.

## In tree today

- Public project scaffold, CLI (`version`, `doctor`, `serve`), and config contract.
- OIDC/JWKS bearer validation on `POST /mcp`.
- Payload-safe audit sink and stable audit schema docs.
- Deterministic `doctor` (config, audit sink; opt-in JWKS, token authorization, and live MCP tool invocation checks).
- SDK-backed MCP Streamable HTTP (`initialize`, `tools/list`, `tools/call`).
- Public-safe remote MCP conformance fixtures.
- Read-only `postgres_select` with group tool packs and table/column allowlists.
- Passthrough result-shaping hook only (not data-level governance).

## Alpha (next)

- Doctor: public ingress or tunnel reachability checks.
- Helm chart as the first production packaging path.

## Beta

- Result filtering implementation.
- OpenAI tunnel deployment profile.
- Anthropic tunnel deployment profile.
- Okta or Entra reference deployment.
- Signed container images.
- SBOM publication.

## Later

- Connector SDK after multiple in-tree connectors prove the shape.
- SAML.
- Terraform modules.
- Policy UI.

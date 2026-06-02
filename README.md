# MCP Data Gateway

A self-hosted MCP gateway that puts your IdP and policy controls in front of internal tools, so hosted AI assistants can use them without moving private systems into a third-party SaaS control plane.

This project is an early open-source scaffold for teams that want to give assistants governed access to internal operational data while keeping identity, policy, audit, and network boundaries under their own control.

## Business Problem

Enterprises want AI assistants to help with real work: checking customer records, reading operational data, preparing account context, and answering internal questions. The useful data usually lives behind private networks, in systems that already have access rules, audit expectations, and compliance obligations.

Most hosted assistants need a public or provider-reachable MCP endpoint. That creates a control problem: teams need the endpoint to be reachable by the assistant, but they cannot let it become an ungoverned bridge into private tools.

MCP Data Gateway is the control point for that gap. It is designed to sit at the customer edge, validate the caller with the customer's IdP, map groups to allowed tool packs, enforce connector allowlists, and emit payload-safe audit events before private data is returned.

## Who This Is For

- Platform and infrastructure teams evaluating hosted AI assistants with internal data access.
- Security teams that need identity, authorization, audit, and data-boundary controls around MCP.
- Data and operations teams that want a narrow, read-only proof path before exposing broader tools.
- Builders who need a reference gateway pattern instead of a one-off MCP server wired directly to a database.

## Why

Hosted AI clients need a reachable MCP endpoint. Enterprises still need to keep identity, policy, audit, network boundaries, and data residency under their control.

The gateway sits between hosted clients and private tools:

```text
Claude / ChatGPT / API client
  -> public ingress or provider tunnel
  -> MCP Data Gateway
  -> OIDC identity validation
  -> group-to-tool-pack policy
  -> audit log
  -> private read-only adapters
  -> governed internal data
```

Provider tunnels improve the transport story by avoiding inbound firewall rules, but they do not replace identity, policy, audit, or result controls. The gateway is the enforcement point behind either public ingress or a tunnel.

This repository is not a hosted service. It is meant to be deployed by the organization that owns the data.

## MVP Scope

Target for the first tagged release (still pre-alpha today):

- OIDC only.
- One gateway per organization or business unit.
- Claude-first remote MCP path.
- Postgres read-only adapter first.
- Helm-first deployment path.
- Group to named tool-pack policy.
- Doctor checks for DNS, OAuth metadata, JWKS, token claims, allowed tool invocation, and denied-user negative tests.
- Stable audit log schema.
- Result-shaping interface, even if the first implementation is passthrough.

**Implemented in this repository (unreleased):** OIDC/JWKS auth on `/mcp`, payload-safe audit, deterministic doctor (config, audit sink, opt-in JWKS/token checks, optional live MCP tool invocation checks), SDK-backed MCP Streamable HTTP, public-safe remote MCP conformance fixtures, read-only `postgres_select` with group tool packs and table/column allowlists, and passthrough result shaping only.

**Not implemented yet:** Helm chart, SBOM/signing, ingress/tunnel reachability doctor checks, and data-level result filtering.

Non-goals for the first release:

- SAML.
- Terraform.
- Generic connector SDK.
- Claims that the gateway fully governs data before column allowlists and result filtering are implemented.
- Prompt-injection prevention inside tool results.
- IdP user lifecycle management.

## Quick Start

```bash
git clone https://github.com/fyne-coder/mcp-data-gateway.git
cd mcp-data-gateway
make ci
go run ./cmd/mcp-data-gateway version
go run ./cmd/mcp-data-gateway doctor --config configs/example.yaml
```

Default doctor validates config and audit sink support without live network calls. Use `--check-jwks`, `--allowed-token-file`, `--denied-token-file`, `--mcp-url`, and `--tool-call-file` for opt-in JWKS, token authorization, and live MCP tool invocation checks. See [Operations](docs/operations.md).

Run the gateway locally:

```bash
go run ./cmd/mcp-data-gateway serve --config configs/example.yaml
```

The server exposes:

- `GET /healthz`
- `GET /readyz`
- `POST /mcp` with OIDC auth, payload-safe audit, SDK-backed MCP protocol handling, and a read-only `postgres_select` tool gated by group tool packs and table/column allowlists

## Design Principles

- Customer-hosted by default.
- Keep the private MCP adapter private.
- Expose only a controlled gateway or tunnel target.
- Treat tunnels as transport, not authorization.
- Enforce policy before tool invocation.
- Provide a result-shaping hook before claiming data-level governance.
- Make audit logs payload-safe, schema-versioned, and SIEM-friendly.
- Ship negative auth tests as a release gate.
- Prefer mature auth, policy, parsing, and crypto libraries over custom implementations.

## Documentation

- [Requirements](docs/requirements.md)
- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [Deployment](docs/deployment.md)
- [Operations](docs/operations.md)
- [Development](docs/development.md)
- [Threat Model](docs/threat-model.md)
- [Audit Log Schema](docs/audit-log-schema.md)
- [Publishing](docs/publishing.md)
- [Roadmap](docs/roadmap.md)

## Status

Pre-alpha and unreleased. The gateway implements OIDC/JWKS auth, payload-safe audit, deterministic doctor checks, SDK-backed MCP Streamable HTTP, public-safe conformance fixtures, and a read-only `postgres_select` tool gated by group tool packs and allowlists. Packaging (Helm), supply-chain signing, and full data governance are not shipped yet.

## License

MIT

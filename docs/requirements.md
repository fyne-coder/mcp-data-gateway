# Requirements

## Product Goal

Build a self-hosted MCP gateway that lets hosted AI clients reach internal tools while the customer keeps identity, policy, network, logs, and data residency under its control.

## Business Context

The project is for organizations that want assistants to use internal operational data without bypassing the controls they already rely on for human and service access. The gateway should make MCP access reviewable by infrastructure, security, and data owners before any broader production rollout.

The first business proof is intentionally narrow: a hosted MCP client reaches a customer-hosted HTTPS gateway, OIDC identifies the caller, group policy selects an allowed tool pack, a read-only Postgres tool returns only allowlisted columns, and the gateway emits payload-safe audit events. The private MCP server or database adapter is not the public endpoint.

## Launch Positioning

A self-hosted MCP gateway that puts your IdP and your policy engine in front of internal tools, so hosted AI assistants can use them without your data leaving your environment.

## Core Requirements

- Support public ingress and provider tunnel edge modes.
- Validate OIDC tokens and JWKS metadata with mature libraries (`go-oidc` in the first slice).
- Reject unauthenticated `/mcp` requests before MCP proxy work begins.
- Map user groups to named tool packs.
- Enforce policy before every tool call.
- Provide a result-shaping hook before returning tool results.
- Emit payload-safe audit events using a stable schema.
- Keep private MCP adapters and databases off the public internet.
- Ship a doctor command that proves deployment readiness. **Current:** config, audit sink, opt-in JWKS/token checks, and optional live MCP tool invocation checks. **Planned:** ingress/tunnel reachability.
- Include negative auth checks for denied users and forbidden tools. **Current:** gateway and doctor token negative paths; tool deny-before-query for disallowed groups, tables, and columns.

## MVP Boundary

Version `0.1.0` is a pre-alpha source release. In-tree today: OIDC on `/mcp`, payload-safe audit, SDK-backed MCP, remote MCP conformance fixtures, read-only Postgres with allowlists, and passthrough result shaping only. Helm, SBOM/signing, and full data-level governance are not done. SAML, Terraform, generic connector SDKs, and broad "works with every assistant" claims remain later work.

## Edge Modes

- `public_ingress`: DNS, TLS, WAF, and gateway are reachable by the hosted client.
- `openai_tunnel`: an OpenAI tunnel client forwards requests to the private gateway.
- `anthropic_tunnel`: an Anthropic tunnel stack forwards requests to the private gateway.
- `private_http`: local or internal-only use during development and private deployments.

Tunnels are transport choices. They do not replace per-user authorization, tool policy, audit logs, or result filtering.

## Public Launch Bar

- Threat model.
- Negative auth tests in CI.
- Stable audit schema.
- Supply-chain basics: pinned images, SBOM, signed release artifacts.
- Upstream bearer-token rotation docs.
- Explicit non-goals and limits.

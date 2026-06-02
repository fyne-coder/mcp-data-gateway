# Deployment

The deployment goal is to make hosted MCP clients reachable without moving private adapters or databases onto the public internet. Current hosted assistant platforms, including Claude and ChatGPT remote-MCP flows, require an externally reachable HTTPS MCP endpoint. The gateway should be the only public or provider-reachable control point, and it should run in infrastructure owned by the organization that owns the data.

## MVP Deployment Shape

```text
Hosted client
  -> public ingress or provider tunnel
  -> MCP Data Gateway
  -> private adapter
  -> read-only data source
```

Do not expose private adapters directly to the internet.

## Public Ingress

Use public ingress when the hosted client needs a reachable HTTPS `/mcp` URL. Do not point that public URL directly at an internal MCP server or private database adapter; terminate it at the gateway so identity, policy, audit, and allowlist checks run first.

Required controls:

- DNS and TLS.
- WAF or rate limiting.
- OIDC validation at the gateway.
- Private upstream adapter.
- Audit sink.
- Negative auth test before go-live.

## Provider Tunnel

Use a provider tunnel when the customer wants outbound-only connectivity.

The tunnel agent should forward to a private gateway listener. The gateway must still validate identity and enforce policy unless the provider supplies a separately verified identity contract that the gateway can trust.

## Private HTTP

Use `private_http` only for local development or internal-only deployments. It is not a production substitute for a gateway with identity and policy.

## Container Image

The repository includes a Dockerfile for an interim container image build. Treat it as a smoke-tested packaging path, not a full release artifact path: pinned image publishing, SBOM generation, and signing are still release-bar items.

## Helm

Helm is the first intended production packaging path, but the chart is not in-tree yet. Terraform modules and other deployment systems should wait until the Kubernetes shape is stable.

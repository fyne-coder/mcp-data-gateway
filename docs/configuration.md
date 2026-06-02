# Configuration

Configuration is YAML. See [configs/example.yaml](../configs/example.yaml).

## Server

```yaml
server:
  listen: "127.0.0.1:8080"
```

For tunnel deployments, prefer a private listener reachable only by the tunnel agent or local sidecar.

## Edge

```yaml
edge:
  mode: "public_ingress"
```

Supported modes:

- `public_ingress`
- `openai_tunnel`
- `anthropic_tunnel`
- `private_http`

Edge mode controls transport. It does not control authorization.

## Auth

```yaml
auth:
  issuer: "https://idp.example.com/oauth2/default"
  audience: "mcp-data-gateway"
  jwks_url: "https://idp.example.com/oauth2/default/v1/keys"
  group_claim_name: "groups"
  required_groups:
    - "mcp-users"
```

| Field | Purpose |
| --- | --- |
| `issuer` | Expected token issuer (`iss`). Used for OIDC discovery when `jwks_url` is empty. |
| `audience` | Expected token audience (`aud`). |
| `jwks_url` | Optional JWKS endpoint override for offline, deterministic test, or static-JWKS deployments. When empty, keys are loaded from the issuer's OIDC discovery document at gateway startup. |
| `group_claim_name` | JWT claim containing group membership. Defaults to `groups`. |
| `required_groups` | Caller must belong to at least one listed group to reach `/mcp`. |

Production auth uses `github.com/coreos/go-oidc/v3` for JWKS signature verification. The gateway expects JWT bearer tokens with OIDC-style `iss`, `aud`, `exp`, `sub`, and group claims; opaque access tokens are not supported in the first slice. It validates issuer, audience, expiry, signature, and group membership before delegating authorized `POST /mcp` traffic to the SDK-backed MCP handler. `/healthz` and `/readyz` stay unauthenticated.

MCP protocol handling uses `github.com/modelcontextprotocol/go-sdk` with a stateless streamable HTTP transport.

## Postgres read-only tool

```yaml
postgres:
  dsn_env: "MCP_DATA_GATEWAY_POSTGRES_DSN"
  max_rows: 100
  tool_packs:
    analyst:
      tables:
        public.example_table:
          columns:
            - id
            - name
```

| Field | Purpose |
| --- | --- |
| `dsn_env` | Name of the environment variable holding the Postgres DSN. Do not commit literal DSNs in YAML. |
| `max_rows` | Upper bound for `postgres_select` row limits. |
| `tool_packs` | Per-pack table and column allowlists used before any query is built. |

The `postgres_select` MCP tool accepts structured input only (`table`, `columns`, optional `limit`). Raw SQL is not accepted. Identifiers are quoted with `pgx.Identifier.Sanitize`; limits are parameterized. Result shaping is a passthrough hook today and does not claim data-level governance.

## Policy

```yaml
policy:
  default_tool_pack: "analyst"
  group_tool_packs:
    mcp-users:
      - "analyst"
```

The MVP policy model is group to named tool pack. This keeps the first release legible and testable.

## Audit

```yaml
audit:
  sink: "stdout"
  include_payloads: false
```

| Field | Purpose |
| --- | --- |
| `sink` | Where newline-delimited JSON audit events are written. Supported values: `stdout` (default), `discard` (tests only). |
| `include_payloads` | Reserved for future controlled diagnostics. Must remain `false` in production; raw tool payloads are never written in the current scaffold. |

`POST /mcp` writes one payload-safe audit event per request. Events include schema version, event ID, timestamp, request ID, edge mode, decision, and result status. Verified actor subject and groups are included on `403` and authorized paths only. `/healthz` and `/readyz` are not audited.

Payload capture is disabled by default. Audit events should remain safe for SIEM forwarding.

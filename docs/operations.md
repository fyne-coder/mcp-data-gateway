# Operations

## Doctor Command

The `doctor` command is the main operational proof path for the current gateway surface.

Default checks (no live network required):

```bash
go run ./cmd/mcp-data-gateway doctor --config configs/example.yaml
```

Output uses stable `PASS`, `SKIP`, and `FAIL` lines for:

- Config validation (`PASS config`).
- Audit sink support (`PASS audit sink`).

Opt-in checks:

- `--check-jwks` fetches `auth.jwks_url` when set, otherwise runs OIDC discovery against `auth.issuer`.
- `--allowed-token-file PATH` verifies a bearer token with the same OIDC verifier as `/mcp` and requires group authorization to succeed.
- `--denied-token-file PATH` verifies a bearer token but requires group authorization to fail.
- `--mcp-url URL` with `--tool-call-file PATH` and token files optionally prove live `POST /mcp` tool invocation:
  - With `--allowed-token-file`, doctor requires HTTP 2xx and protocol success (no JSON-RPC `error`, no `result.isError`). Emits `PASS allowed tool invocation`.
  - With `--denied-token-file`, doctor requires HTTP 401/403, or HTTP 2xx without protocol success. Emits `PASS denied tool invocation`.
  - Use `docs/examples/postgres_select_tools_call.json` for an allowlisted `postgres_select` body matching `configs/example.yaml`.

Token files must contain only the bearer token. Keep them owned by the operator running doctor and mode `0600`. Do not pass raw tokens as CLI arguments. Doctor output never includes token values, claim payloads, JSON-RPC bodies, or response bodies.

Example with placeholders (replace URL and token paths before running):

```bash
export MCP_GATEWAY_URL="https://gateway.example.com"
go run ./cmd/mcp-data-gateway doctor --config configs/example.yaml \
  --check-jwks \
  --mcp-url "${MCP_GATEWAY_URL}/mcp" \
  --tool-call-file docs/examples/postgres_select_tools_call.json \
  --allowed-token-file /path/to/allowed.jwt \
  --denied-token-file /path/to/denied.jwt
```

The example config uses placeholder IdP URLs. `--check-jwks` requires a reachable `auth.jwks_url` or issuer discovery endpoint and will fail against the placeholder until you replace it.

Hosted-client conformance checks (`initialize`, `tools/list`, `tools/call`, error paths) are documented in [MCP conformance fixture](examples/mcp-conformance.md).

Future doctor checks (not implemented yet):

- Public ingress or tunnel reachability.
- Audit event emission smoke tests beyond sink support.

## Logs

Logs must avoid raw tool payloads by default. Use request IDs and audit event IDs for correlation.

## MCP protocol surface

Authorized `POST /mcp` requests are handled by the official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`) using a stateless streamable HTTP handler with JSON responses. Auth and audit run in the gateway before protocol delegation.

Current behavior:

- `initialize` succeeds and advertises server name `mcp-data-gateway`.
- `tools/list` includes the read-only `postgres_select` tool when the gateway is configured with Postgres allowlists.
- `tools/call` for `postgres_select` runs only after group tool-pack policy and table/column allowlist checks; disallowed callers or inputs do not reach the database layer.
- Set `MCP_DATA_GATEWAY_POSTGRES_DSN` (or the configured `postgres.dsn_env` name) before expecting live query results from `serve`.

## Audit

Audit events are newline-delimited JSON. See [Audit Log Schema](audit-log-schema.md). Auth and group checks still emit one payload-safe event per `POST /mcp` request before or after SDK handling; raw tokens, request bodies, tool arguments, and tool results are never logged.

## Credential Rotation

Production deployments need documented rotation for:

- OIDC client credentials, when used.
- Upstream bearer tokens.
- Tunnel credentials.
- Connector credentials.

## Incident Checklist

1. Disable the affected edge mode or ingress route.
2. Rotate upstream bearer credentials.
3. Preserve audit logs.
4. Identify affected subjects, tool packs, and connectors.
5. Patch policy or connector allowlists.
6. Run negative auth tests before restoring access.

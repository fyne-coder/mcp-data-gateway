# Remote MCP Conformance Fixture

Public-safe, parameterized checks for hosted MCP clients (Claude Desktop, Cursor, and similar) against a deployed gateway. Replace placeholders with your deployment values. Do not commit real URLs, tokens, DSNs, or hostnames.

## Prerequisites

- Gateway running with Postgres allowlists configured (see `configs/example.yaml` for table/column shape).
- `MCP_DATA_GATEWAY_POSTGRES_DSN` set on the gateway host when exercising live queries.
- Bearer token files owned by the operator, mode `0600`, containing only the raw JWT.

## Environment variables

| Variable | Purpose |
| --- | --- |
| `MCP_GATEWAY_URL` | Base URL of the gateway (e.g. `https://gateway.example.com`) |
| `MCP_ALLOWED_TOKEN_FILE` | Path to JWT that verifies and passes group authorization |
| `MCP_DENIED_TOKEN_FILE` | Path to JWT that verifies but fails group authorization |

## JSON-RPC fixtures

| File | Method | Expected outcome (allowed token) |
| --- | --- | --- |
| `docs/examples/mcp_initialize.json` | `initialize` | HTTP 2xx, JSON-RPC success, no `result.isError` |
| `docs/examples/mcp_tools_list.json` | `tools/list` | HTTP 2xx, response includes `postgres_select` |
| `docs/examples/postgres_select_tools_call.json` | `tools/call` | HTTP 2xx, JSON-RPC success for allowlisted table/columns |
| `docs/examples/mcp_tools_call_forbidden_table.json` | `tools/call` | HTTP 2xx with JSON-RPC error or `result.isError`, or HTTP 403; must not return rows for forbidden table |

## Doctor (operator smoke)

Doctor aggregates config, JWKS, token, and optional live tool checks. It never prints tokens, claim payloads, request bodies, response bodies, or DSNs.

```bash
export MCP_GATEWAY_URL="https://gateway.example.com"

go run ./cmd/mcp-data-gateway doctor --config /path/to/production.yaml \
  --check-jwks \
  --allowed-token-file "$MCP_ALLOWED_TOKEN_FILE" \
  --denied-token-file "$MCP_DENIED_TOKEN_FILE" \
  --mcp-url "${MCP_GATEWAY_URL}/mcp" \
  --tool-call-file docs/examples/postgres_select_tools_call.json
```

Expected lines when all inputs are valid:

- `PASS allowed tool invocation`
- `PASS denied tool invocation`

## curl (manual client conformance)

Post each fixture with `Authorization: Bearer $(cat "$MCP_ALLOWED_TOKEN_FILE")` and `Content-Type: application/json`. Classify outcomes the same way as audit: HTTP 2xx with top-level JSON-RPC `error` or `result.isError` is a protocol error, not success.

```bash
curl -sS -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer $(cat "$MCP_ALLOWED_TOKEN_FILE")" \
  -H "Content-Type: application/json" \
  --data-binary @"docs/examples/mcp_initialize.json" \
  "${MCP_GATEWAY_URL}/mcp"
```

Repeat for `mcp_tools_list.json` and `postgres_select_tools_call.json`. For the denied-user negative path, use `"$MCP_DENIED_TOKEN_FILE"` with `postgres_select_tools_call.json` and expect HTTP 401/403 or protocol-level denial, not a successful tool result.

## Error-path checks

1. **Denied bearer**: same `tools/call` body with denied token file; must not get protocol success.
2. **Forbidden table**: `mcp_tools_call_forbidden_table.json` with allowed token; must not query disallowed table (gateway returns error before database).
3. **Missing auth**: omit `Authorization`; expect HTTP 401 and no tool execution.

Record pass/fail per check in your private run log; keep tokens and response payloads out of public artifacts.

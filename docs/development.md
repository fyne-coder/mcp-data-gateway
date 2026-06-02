# Development

## Prerequisites

- Go 1.25 or newer.
- `make`.

## Local Loop

```bash
make ci
go run ./cmd/mcp-data-gateway doctor --config configs/example.yaml
go run ./cmd/mcp-data-gateway serve --config configs/example.yaml
```

Default `doctor` is network-free: it validates config and audit sink support, then skips JWKS and token checks unless you opt in.

```bash
# Optional JWKS reachability against configured auth.jwks_url or OIDC discovery
go run ./cmd/mcp-data-gateway doctor --config configs/example.yaml --check-jwks

# Optional token checks read tokens from files (never pass raw tokens on the CLI)
go run ./cmd/mcp-data-gateway doctor --config configs/example.yaml \
  --allowed-token-file /path/to/allowed.jwt \
  --denied-token-file /path/to/denied.jwt

# Optional live tool invocation (requires reachable gateway and token files)
export MCP_GATEWAY_URL="https://gateway.example.com"
go run ./cmd/mcp-data-gateway doctor --config configs/example.yaml \
  --mcp-url "${MCP_GATEWAY_URL}/mcp" \
  --tool-call-file docs/examples/postgres_select_tools_call.json \
  --allowed-token-file /path/to/allowed.jwt \
  --denied-token-file /path/to/denied.jwt
```

Doctor never prints bearer tokens, token claims, JSON-RPC request bodies, response bodies, or DSNs.

Remote MCP conformance fixtures for hosted clients live under `docs/examples/`; see [MCP conformance fixture](examples/mcp-conformance.md).

## Repository Gate

`make ci` is the canonical local gate. It runs:

- `gofmt` check
- `go vet`
- `go test ./...`
- `go build -o bin/mcp-data-gateway ./cmd/mcp-data-gateway`

## Dependency Rule

Use mature maintained libraries or official SDKs for standard hard problems:

- OIDC and JWKS validation.
- OAuth metadata discovery.
- MCP protocol handling.
- Policy engines.
- Database access.
- Cryptographic signing and verification.
- Tunnel provider clients.

Hand-rolled logic is acceptable for small orchestration glue, typed interfaces, test scaffolds, and placeholder behavior while the project is pre-alpha. Any production auth, crypto, policy, protocol, or database implementation must document the selected library and why it fits.

## Test Expectations

- Unit tests for config, policy, gateway handlers, and audit schema.
- Negative auth tests for denied users and forbidden tools.

Focused auth, audit, and MCP protocol checks:

```bash
go test ./internal/audit ./internal/gateway ./internal/mcpserver
go test ./internal/auth ./internal/gateway ./internal/config
```

Authorized `/mcp` tests post JSON-RPC messages through the gateway with `auth.FakeVerifier` and `postgres.FakeQuerier`. They assert SDK-backed `initialize`, `tools/list` including `postgres_select`, allowlisted `tools/call` success, deny-before-query failures, and audit events that exclude tool arguments and row values.

Gateway tests use `auth.FakeVerifier` and `audit.MemorySink` so unit tests do not call a live IdP or write to stdout. Auth package tests pin JWKS with `httptest` when exercising `go-oidc`.
- Integration tests for each supported edge mode.
- Conformance fixtures for each supported hosted client.

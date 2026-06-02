# Audit Log Schema

Audit events are newline-delimited JSON. Raw tool arguments and results are excluded by default.

```json
{
  "schema_version": "mcp-data-gateway.audit.v1",
  "event_id": "01J00000000000000000000000",
  "timestamp": "2026-06-01T16:15:00-04:00",
  "request_id": "req_123",
  "actor": {
    "subject": "user@example.com",
    "groups": ["mcp-users"]
  },
  "client": {
    "name": "claude",
    "edge_mode": "public_ingress"
  },
  "decision": {
    "allowed": true,
    "tool_pack": "analyst",
    "tool": "mcp"
  },
  "result": {
    "status": "ok",
    "redaction_applied": true
  }
}
```

## Contract

- `schema_version` is required.
- `event_id`, `timestamp`, and `request_id` are required.
- `decision.allowed` is required for every tool call.
- `decision.tool` is `mcp` for the gateway placeholder path in this slice.
- Payload fields must be absent unless explicitly enabled for a controlled diagnostic run.
- Raw bearer tokens, request bodies, tool arguments, and tool results are never recorded.
- `result.redaction_applied` is `true` in this slice because the gateway emits a payload-safe event and no raw payload is eligible for storage.
- `POST /mcp` emits exactly one audit event per request for auth decision paths.
- `/healthz` and `/readyz` do not emit audit events.

## Request correlation

- When the client sends `X-Request-Id`, the gateway preserves it on the response and in the audit event.
- When `X-Request-Id` is absent, the gateway generates a non-empty `request_id`.

## Result status values (auth slice)

| `result.status` | HTTP | `decision.allowed` |
| --- | --- | --- |
| `unauthorized` | 401 | `false` |
| `forbidden` | 403 | `false` |
| `ok` | 2xx with successful JSON-RPC response (no top-level `error`, no `result.isError`) | `true` |
| `error` | non-2xx from MCP handler, or 2xx with JSON-RPC `error` or MCP `result.isError` in the response body | `true` |

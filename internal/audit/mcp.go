package audit

import (
	"bytes"
	"encoding/json"
	"time"
)

const (
	ResultUnauthorized    = "unauthorized"
	ResultForbidden       = "forbidden"
	ResultOK              = "ok"
	ResultError           = "error"
	ResultNotImplemented  = "not_implemented" // deprecated; retained for schema docs migration
	ClientNamePlaceholder = "mcp-client"
	ToolMCPPlaceholder    = "mcp"
)

// MCPResultStatus maps HTTP status and an optional MCP response body to audit
// result.status. Response bytes are used only for classification and must not be
// stored in audit events.
func MCPResultStatus(httpStatus int, responseBody []byte) string {
	if httpStatus < 200 || httpStatus >= 300 {
		return ResultError
	}
	if hasJSONRPCError(responseBody) {
		return ResultError
	}
	return ResultOK
}

func hasJSONRPCError(body []byte) bool {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return false
	}
	var env struct {
		Error  json.RawMessage `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return false
	}
	if len(env.Error) > 0 && !bytes.Equal(env.Error, []byte("null")) {
		return true
	}
	if len(env.Result) == 0 || bytes.Equal(env.Result, []byte("null")) {
		return false
	}
	var result struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(env.Result, &result); err != nil {
		return false
	}
	return result.IsError
}

// MCPEvent builds a payload-safe audit event for a POST /mcp auth decision.
func MCPEvent(requestID, eventID, edgeMode, toolPack string, actor Actor, allowed bool, status string) Event {
	decision := Decision{Allowed: allowed, Tool: ToolMCPPlaceholder}
	if allowed && toolPack != "" {
		decision.ToolPack = toolPack
	}
	return Event{
		SchemaVersion: SchemaVersion,
		EventID:       eventID,
		Timestamp:     time.Now().UTC(),
		RequestID:     requestID,
		Actor:         actor,
		Client: Client{
			Name:     ClientNamePlaceholder,
			EdgeMode: edgeMode,
		},
		Decision: decision,
		Result: Result{
			Status:           status,
			RedactionApplied: true,
		},
	}
}

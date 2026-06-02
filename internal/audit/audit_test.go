package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fyne-coder/mcp-data-gateway/internal/config"
)

func TestWriteAddsSchemaVersion(t *testing.T) {
	var out bytes.Buffer
	if err := Write(&out, Event{EventID: "evt_1", RequestID: "req_1"}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	var event Event
	if err := json.Unmarshal(out.Bytes(), &event); err != nil {
		t.Fatalf("audit event is not JSON: %v", err)
	}
	if event.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %q, want %q", event.SchemaVersion, SchemaVersion)
	}
}

func TestRequestIDPreservesInboundHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-Request-Id", "req-inbound")
	if got := RequestID(req); got != "req-inbound" {
		t.Fatalf("RequestID = %q, want %q", got, "req-inbound")
	}
}

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	got := RequestID(req)
	if got == "" {
		t.Fatal("RequestID is empty")
	}
	if !strings.HasPrefix(got, "req_") {
		t.Fatalf("RequestID = %q, want req_ prefix", got)
	}
}

func TestNewEventIDIsNonEmpty(t *testing.T) {
	if got := NewEventID(); got == "" {
		t.Fatal("NewEventID returned empty string")
	}
}

func TestMCPResultStatusHTTPError(t *testing.T) {
	if got := MCPResultStatus(http.StatusInternalServerError, nil); got != ResultError {
		t.Fatalf("status = %q, want %q", got, ResultError)
	}
}

func TestMCPResultStatusJSONRPCErrorOverHTTP200(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"unknown tool"}}`)
	if got := MCPResultStatus(http.StatusOK, body); got != ResultError {
		t.Fatalf("status = %q, want %q", got, ResultError)
	}
}

func TestMCPResultStatusJSONRPCSuccessOverHTTP200(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05"}}`)
	if got := MCPResultStatus(http.StatusOK, body); got != ResultOK {
		t.Fatalf("status = %q, want %q", got, ResultOK)
	}
}

func TestMCPResultStatusMCPToolErrorOverHTTP200(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"denied"}],"isError":true}}`)
	if got := MCPResultStatus(http.StatusOK, body); got != ResultError {
		t.Fatalf("status = %q, want %q", got, ResultError)
	}
}

func TestMCPResultStatusEmptyBodyOverHTTP200(t *testing.T) {
	if got := MCPResultStatus(http.StatusOK, nil); got != ResultOK {
		t.Fatalf("status = %q, want %q", got, ResultOK)
	}
}

func TestMCPEventShape(t *testing.T) {
	event := MCPEvent("req_1", "evt_1", "public_ingress", "analyst", Actor{Subject: "user-1", Groups: []string{"mcp-users"}}, true, ResultOK)
	if event.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", event.SchemaVersion, SchemaVersion)
	}
	if event.Client.EdgeMode != "public_ingress" {
		t.Fatalf("client.edge_mode = %q, want public_ingress", event.Client.EdgeMode)
	}
	if !event.Decision.Allowed {
		t.Fatal("decision.allowed = false, want true")
	}
	if event.Decision.Tool != ToolMCPPlaceholder {
		t.Fatalf("decision.tool = %q, want %q", event.Decision.Tool, ToolMCPPlaceholder)
	}
	if event.Result.Status != ResultOK {
		t.Fatalf("result.status = %q, want %q", event.Result.Status, ResultOK)
	}
	if !event.Result.RedactionApplied {
		t.Fatal("result.redaction_applied = false, want true")
	}
}

func TestNewSinkStdout(t *testing.T) {
	sink, err := NewSink(config.AuditConfig{Sink: "stdout"})
	if err != nil {
		t.Fatalf("NewSink returned error: %v", err)
	}
	if _, ok := sink.(WriterSink); !ok {
		t.Fatalf("sink type = %T, want WriterSink", sink)
	}
}

func TestMemorySinkRecordsEvents(t *testing.T) {
	sink := &MemorySink{}
	if err := sink.Emit(Event{EventID: "evt_1"}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if sink.Len() != 1 {
		t.Fatalf("events = %d, want 1", sink.Len())
	}
}

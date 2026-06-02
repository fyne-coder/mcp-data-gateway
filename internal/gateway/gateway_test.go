package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fyne-coder/mcp-data-gateway/internal/audit"
	"github.com/fyne-coder/mcp-data-gateway/internal/auth"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/fyne-coder/mcp-data-gateway/internal/postgres"
)

func testConfig() config.Config {
	return config.Config{
		Edge: config.EdgeConfig{Mode: config.EdgePublicIngress},
		Auth: config.AuthConfig{
			RequiredGroups: []string{"mcp-users"},
		},
		Policy: config.PolicyConfig{
			DefaultToolPack: "analyst",
			GroupToolPacks: map[string][]string{
				"mcp-users": {"analyst"},
			},
		},
		Postgres: config.PostgresConfig{
			DSNEnv:  "MCP_DATA_GATEWAY_POSTGRES_DSN",
			MaxRows: 100,
			ToolPacks: map[string]config.ToolPackAllowlist{
				"analyst": {
					Tables: map[string]config.TableAllowlist{
						"public.example_table": {Columns: []string{"id", "name"}},
					},
				},
			},
		},
	}
}

func testServer(verifier auth.Verifier) (Server, *audit.MemorySink) {
	return testServerWithQuerier(verifier, &postgres.FakeQuerier{})
}

func testServerWithQuerier(verifier auth.Verifier, querier *postgres.FakeQuerier) (Server, *audit.MemorySink) {
	sink := &audit.MemorySink{}
	return NewWithDeps(testConfig(), verifier, sink, querier), sink
}

func postMCP(t *testing.T, server Server, authHeader, requestID string) *httptest.ResponseRecorder {
	return postMCPMessage(t, server, authHeader, requestID, "")
}

func postMCPMessage(t *testing.T, server Server, authHeader, requestID, message string) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if message != "" {
		body = strings.NewReader(message)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	if message != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}

func authorizedMCP(t *testing.T, server Server, message string) *httptest.ResponseRecorder {
	t.Helper()
	return postMCPMessage(t, server, "Bearer valid-token", "", message)
}

func requireOneAuditEvent(t *testing.T, sink *audit.MemorySink) audit.Event {
	t.Helper()
	if sink.Len() != 1 {
		t.Fatalf("audit events = %d, want 1", sink.Len())
	}
	return sink.Events[0]
}

func assertAuditShape(t *testing.T, event audit.Event, allowed bool, status string) {
	t.Helper()
	if event.SchemaVersion != audit.SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", event.SchemaVersion, audit.SchemaVersion)
	}
	if event.EventID == "" {
		t.Fatal("event_id is empty")
	}
	if event.RequestID == "" {
		t.Fatal("request_id is empty")
	}
	if event.Timestamp.IsZero() {
		t.Fatal("timestamp is zero")
	}
	if event.Client.EdgeMode != config.EdgePublicIngress {
		t.Fatalf("client.edge_mode = %q, want %q", event.Client.EdgeMode, config.EdgePublicIngress)
	}
	if event.Decision.Allowed != allowed {
		t.Fatalf("decision.allowed = %v, want %v", event.Decision.Allowed, allowed)
	}
	if event.Result.Status != status {
		t.Fatalf("result.status = %q, want %q", event.Result.Status, status)
	}
	if event.Decision.Tool != audit.ToolMCPPlaceholder {
		t.Fatalf("decision.tool = %q, want %q", event.Decision.Tool, audit.ToolMCPPlaceholder)
	}
}

func assertNoSensitivePayload(t *testing.T, raw []byte, secrets ...string) {
	t.Helper()
	text := string(raw)
	for _, secret := range secrets {
		if secret != "" && strings.Contains(text, secret) {
			t.Fatalf("audit event contains sensitive value %q", secret)
		}
	}
	for _, forbidden := range []string{"Bearer ", "tool_args", "tool_results", "request_body"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("audit event contains forbidden field marker %q", forbidden)
		}
	}
}

func TestHandlerHealthzUnauthenticated(t *testing.T) {
	server, sink := testServer(auth.FakeVerifier{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if sink.Len() != 0 {
		t.Fatalf("healthz wrote %d audit events, want 0", sink.Len())
	}
}

func TestHandlerReadyzUnauthenticated(t *testing.T) {
	server, sink := testServer(auth.FakeVerifier{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if sink.Len() != 0 {
		t.Fatalf("readyz wrote %d audit events, want 0", sink.Len())
	}
}

func TestHandlerMCPMissingAuthorization(t *testing.T) {
	server, sink := testServer(auth.FakeVerifier{})
	rec := postMCP(t, server, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, false, audit.ResultUnauthorized)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw)
}

func TestHandlerMCPMalformedAuthorization(t *testing.T) {
	server, sink := testServer(auth.FakeVerifier{})
	rec := postMCP(t, server, "Token abc", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, false, audit.ResultUnauthorized)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "abc")
}

func TestHandlerMCPInvalidToken(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{}}
	server, sink := testServer(verifier)
	rec := postMCP(t, server, "Bearer unknown", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, false, audit.ResultUnauthorized)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "unknown")
}

func TestHandlerMCPForbiddenMissingGroup(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"other-group"}},
	}}
	server, sink := testServer(verifier)
	rec := postMCP(t, server, "Bearer valid-token", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, false, audit.ResultForbidden)
	if event.Actor.Subject != "user-1" {
		t.Fatalf("actor.subject = %q, want %q", event.Actor.Subject, "user-1")
	}
	if len(event.Actor.Groups) != 1 || event.Actor.Groups[0] != "other-group" {
		t.Fatalf("actor.groups = %v, want [other-group]", event.Actor.Groups)
	}
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "valid-token")
}

func TestHandlerMCPInitializeWithRequiredGroup(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServer(verifier)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"name":"mcp-data-gateway"`) {
		t.Fatalf("response missing server name: %s", text)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, true, audit.ResultOK)
	if event.Actor.Subject != "user-1" {
		t.Fatalf("actor.subject = %q, want %q", event.Actor.Subject, "user-1")
	}
	if event.Decision.ToolPack != "analyst" {
		t.Fatalf("decision.tool_pack = %q, want %q", event.Decision.ToolPack, "analyst")
	}
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "valid-token", text)
}

func TestHandlerMCPToolsListIncludesPostgresSelect(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServer(verifier)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"postgres_select"`) {
		t.Fatalf("response missing postgres_select tool: %s", text)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, true, audit.ResultOK)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "valid-token", text)
}

func TestHandlerMCPToolsCallPostgresSelectAllowed(t *testing.T) {
	fake := &postgres.FakeQuerier{
		Rows: []map[string]any{{"id": 1, "name": "example"}},
	}
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServerWithQuerier(verifier, fake)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.example_table","columns":["id","name"],"limit":10}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, `"error"`) {
		t.Fatalf("tools/call returned error: %s", text)
	}
	if fake.QueryCount != 1 {
		t.Fatalf("QueryCount = %d, want 1", fake.QueryCount)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, true, audit.ResultOK)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "valid-token", "public.example_table", "example", text)
}

func TestHandlerMCPToolsCallPostgresSelectForbiddenAuthDoesNotQuery(t *testing.T) {
	fake := &postgres.FakeQuerier{}
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"other-group"}},
	}}
	server, _ := testServerWithQuerier(verifier, fake)
	rec := postMCPMessage(t, server, "Bearer valid-token", "", `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.example_table","columns":["id"]}}}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
}

func TestHandlerMCPToolsCallPostgresSelectDeniedToolPackDoesNotQuery(t *testing.T) {
	fake := &postgres.FakeQuerier{}
	cfg := testConfig()
	cfg.Auth.RequiredGroups = []string{"mcp-users", "contractors"}
	cfg.Policy.GroupToolPacks = map[string][]string{"mcp-users": {"analyst"}}
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"contractors"}},
	}}
	sink := &audit.MemorySink{}
	server := NewWithDeps(cfg, verifier, sink, fake)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.example_table","columns":["id"]}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
}

func TestHandlerMCPToolsCallPostgresSelectDeniedTableDoesNotQuery(t *testing.T) {
	fake := &postgres.FakeQuerier{}
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServerWithQuerier(verifier, fake)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.secret_table","columns":["id"]}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
	event := requireOneAuditEvent(t, sink)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "valid-token", "secret_table")
}

func TestHandlerMCPToolsCallUnavailableTool(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServer(verifier)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"private-query","arguments":{}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"error"`) || !strings.Contains(text, "unknown tool") {
		t.Fatalf("response missing safe tool error: %s", text)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, true, audit.ResultError)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "valid-token", "private-query", text)
}

func TestHandlerMCPJSONRPCErrorOverHTTP200AuditsError(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServer(verifier)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.secret_table","columns":["id"]}}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `"isError":true`) && !strings.Contains(text, `"error"`) {
		t.Fatalf("response missing MCP protocol error: %s", text)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, true, audit.ResultError)
	raw, _ := json.Marshal(event)
	assertNoSensitivePayload(t, raw, "valid-token", "secret_table", text)
}

func TestHandlerMCPSuccessfulJSONRPCOverHTTP200AuditsOK(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServer(verifier)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":10,"method":"initialize","params":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, `"error"`) {
		t.Fatalf("initialize returned JSON-RPC error: %s", text)
	}
	event := requireOneAuditEvent(t, sink)
	assertAuditShape(t, event, true, audit.ResultOK)
}

func TestHandlerMCPAuthorizedWritesOneAuditEvent(t *testing.T) {
	verifier := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		"valid-token": {Subject: "user-1", Groups: []string{"mcp-users"}},
	}}
	server, sink := testServer(verifier)
	rec := authorizedMCP(t, server, `{"jsonrpc":"2.0","id":4,"method":"initialize","params":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if sink.Len() != 1 {
		t.Fatalf("audit events = %d, want 1", sink.Len())
	}
}

func TestHandlerMCPPreservesRequestID(t *testing.T) {
	server, sink := testServer(auth.FakeVerifier{})
	rec := postMCP(t, server, "", "req-client-123")
	if got := rec.Header().Get("X-Request-Id"); got != "req-client-123" {
		t.Fatalf("response X-Request-Id = %q, want %q", got, "req-client-123")
	}
	event := requireOneAuditEvent(t, sink)
	if event.RequestID != "req-client-123" {
		t.Fatalf("request_id = %q, want %q", event.RequestID, "req-client-123")
	}
}

func TestHandlerMCPGeneratesRequestID(t *testing.T) {
	server, sink := testServer(auth.FakeVerifier{})
	rec := postMCP(t, server, "", "")
	event := requireOneAuditEvent(t, sink)
	if event.RequestID == "" {
		t.Fatal("request_id is empty")
	}
	if got := rec.Header().Get("X-Request-Id"); got != event.RequestID {
		t.Fatalf("response X-Request-Id = %q, want %q", got, event.RequestID)
	}
}

// TestAuthEnforcementRequired fails if /mcp skips bearer validation.
func TestAuthEnforcementRequired(t *testing.T) {
	server, _ := testServer(auth.FakeVerifier{Err: auth.ErrUnauthorized})
	rec := postMCP(t, server, "Bearer any", "")
	if rec.Code == http.StatusOK {
		t.Fatal("/mcp returned 200 without auth enforcement")
	}
}

// TestAuditEnforcementRequired fails if /mcp skips audit writes.
func TestAuditEnforcementRequired(t *testing.T) {
	sink := audit.DiscardSink{}
	server := NewWithVerifierAndSink(testConfig(), auth.FakeVerifier{}, sink)
	rec := postMCP(t, server, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	memSink := &audit.MemorySink{}
	server = NewWithVerifierAndSink(testConfig(), auth.FakeVerifier{}, memSink)
	_ = postMCP(t, server, "", "")
	if memSink.Len() != 1 {
		t.Fatalf("audit events = %d, want 1; gateway skipped audit write", memSink.Len())
	}
}

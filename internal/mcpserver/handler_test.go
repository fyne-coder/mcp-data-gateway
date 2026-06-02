package mcpserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fyne-coder/mcp-data-gateway/internal/auth"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/fyne-coder/mcp-data-gateway/internal/policy"
	"github.com/fyne-coder/mcp-data-gateway/internal/postgres"
	"github.com/fyne-coder/mcp-data-gateway/internal/resultshape"
)

func testDeps(fake *postgres.FakeQuerier) Deps {
	return Deps{
		Policy: policy.Engine{
			DefaultToolPack: "analyst",
			GroupToolPacks:  map[string][]string{"mcp-users": {"analyst"}},
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
		Querier: fake,
		Shaper:  resultshape.Passthrough{},
	}
}

func TestStreamableHTTPHandlerInitialize(t *testing.T) {
	handler := StreamableHTTPHandler(testDeps(&postgres.FakeQuerier{}))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `"name":"mcp-data-gateway"`) {
		t.Fatalf("response missing server name: %s", text)
	}
}

func TestToolsListIncludesPostgresSelect(t *testing.T) {
	handler := StreamableHTTPHandler(testDeps(&postgres.FakeQuerier{}))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"postgres_select"`) {
		t.Fatalf("tools/list missing postgres_select: %s", raw)
	}
}

func TestPostgresSelectAllowedActorQueries(t *testing.T) {
	fake := &postgres.FakeQuerier{
		Rows: []map[string]any{{"id": 1, "name": "example"}},
	}
	handler := StreamableHTTPHandler(testDeps(fake))
	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.example_table","columns":["id","name"],"limit":10}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req = req.WithContext(WithInvocation(context.Background(), Invocation{
		Actor: auth.Actor{Subject: "user-1", Groups: []string{"mcp-users"}},
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	raw, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, `"error"`) {
		t.Fatalf("tools/call returned error: %s", text)
	}
	if fake.QueryCount != 1 {
		t.Fatalf("QueryCount = %d, want 1", fake.QueryCount)
	}
	if !strings.Contains(text, `"name":"example"`) {
		t.Fatalf("response missing row data: %s", text)
	}
}

func TestPostgresSelectDeniedGroupDoesNotQuery(t *testing.T) {
	fake := &postgres.FakeQuerier{}
	handler := StreamableHTTPHandler(testDeps(fake))
	body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.example_table","columns":["id"]}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req = req.WithContext(WithInvocation(context.Background(), Invocation{
		Actor: auth.Actor{Subject: "user-2", Groups: []string{"blocked"}},
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
}

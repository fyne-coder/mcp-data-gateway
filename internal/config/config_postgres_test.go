package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAcceptsExamplePostgresAllowlist(t *testing.T) {
	cfg, err := Load("../../configs/example.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Postgres.DSNEnv != "MCP_DATA_GATEWAY_POSTGRES_DSN" {
		t.Fatalf("dsn_env = %q", cfg.Postgres.DSNEnv)
	}
	cols := cfg.Postgres.ToolPacks["analyst"].Tables["public.example_table"].Columns
	if len(cols) != 2 || cols[0] != "id" || cols[1] != "name" {
		t.Fatalf("columns = %v", cols)
	}
}

func TestLoadRejectsEmptyAllowlistColumns(t *testing.T) {
	path := writeConfig(t, `
server:
  listen: "127.0.0.1:8080"
edge:
  mode: "private_http"
auth:
  issuer: "https://idp.example.com"
  audience: "mcp-data-gateway"
  required_groups: ["mcp-users"]
policy:
  default_tool_pack: "analyst"
  group_tool_packs:
    mcp-users: ["analyst"]
postgres:
  dsn_env: "MCP_DATA_GATEWAY_POSTGRES_DSN"
  max_rows: 100
  tool_packs:
    analyst:
      tables:
        public.example_table:
          columns: []
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "columns") {
		t.Fatalf("Load err = %v, want columns validation error", err)
	}
}

func TestLoadRejectsPolicyToolPackMismatch(t *testing.T) {
	path := writeConfig(t, `
server:
  listen: "127.0.0.1:8080"
edge:
  mode: "private_http"
auth:
  issuer: "https://idp.example.com"
  audience: "mcp-data-gateway"
  required_groups: ["mcp-users"]
policy:
  default_tool_pack: "analyst"
  group_tool_packs:
    mcp-users: ["missing-pack"]
postgres:
  dsn_env: "MCP_DATA_GATEWAY_POSTGRES_DSN"
  max_rows: 100
  tool_packs:
    analyst:
      tables:
        public.example_table:
          columns: ["id"]
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "unknown postgres tool pack") {
		t.Fatalf("Load err = %v, want unknown pack error", err)
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

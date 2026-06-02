package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	cfg, err := Load("../../configs/example.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Edge.Mode != EdgePublicIngress {
		t.Fatalf("edge mode = %q, want %q", cfg.Edge.Mode, EdgePublicIngress)
	}
}

func TestLoadRejectsMissingRequiredGroups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := []byte(`
server:
  listen: "127.0.0.1:8080"
edge:
  mode: "private_http"
auth:
  issuer: "https://idp.example.com"
  audience: "mcp-data-gateway"
policy:
  default_tool_pack: "analyst"
  group_tool_packs:
    mcp-users: ["analyst"]
`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load succeeded, want required_groups error")
	}
}

func TestLoadRejectsInvalidEdgeMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := []byte(`
server:
  listen: "127.0.0.1:8080"
edge:
  mode: "raw_internet"
auth:
  issuer: "https://idp.example.com"
  audience: "mcp-data-gateway"
policy:
  default_tool_pack: "analyst"
  group_tool_packs:
    mcp-users: ["analyst"]
`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load succeeded, want invalid edge mode error")
	}
}

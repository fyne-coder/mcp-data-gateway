package policy

import "testing"

func TestResolveToolPackFromMappedGroup(t *testing.T) {
	engine := Engine{
		DefaultToolPack: "analyst",
		GroupToolPacks:  map[string][]string{"mcp-users": {"analyst"}},
	}

	pack, ok := engine.ResolveToolPack(Actor{Subject: "user@example.com", Groups: []string{"mcp-users"}})
	if !ok || pack != "analyst" {
		t.Fatalf("ResolveToolPack() = %q, %v; want analyst, true", pack, ok)
	}
}

func TestResolveToolPackDeniesUnmappedGroup(t *testing.T) {
	engine := Engine{
		DefaultToolPack: "analyst",
		GroupToolPacks:  map[string][]string{"mcp-users": {"analyst"}},
	}

	_, ok := engine.ResolveToolPack(Actor{Subject: "blocked@example.com", Groups: []string{"blocked"}})
	if ok {
		t.Fatal("ResolveToolPack() allowed unmapped group")
	}
}

func TestDecideAllowsPostgresSelect(t *testing.T) {
	engine := Engine{
		DefaultToolPack: "analyst",
		GroupToolPacks:  map[string][]string{"mcp-users": {"analyst"}},
	}

	decision := engine.Decide(Request{
		Actor: Actor{Subject: "user@example.com", Groups: []string{"mcp-users"}},
		Tool:  ToolPostgresSelect,
	})
	if !decision.Allowed || decision.ToolPack != "analyst" {
		t.Fatalf("decision = %#v, want allowed analyst pack", decision)
	}
}

func TestDecideDeniesUnmappedGroup(t *testing.T) {
	engine := Engine{
		DefaultToolPack: "analyst",
		GroupToolPacks:  map[string][]string{"mcp-users": {"analyst"}},
	}

	decision := engine.Decide(Request{
		Actor: Actor{Subject: "blocked@example.com", Groups: []string{"blocked"}},
		Tool:  ToolPostgresSelect,
	})
	if decision.Allowed {
		t.Fatalf("decision allowed request: %#v", decision)
	}
}

func TestDecideDeniesEmptyTool(t *testing.T) {
	engine := Engine{
		DefaultToolPack: "analyst",
		GroupToolPacks:  map[string][]string{"mcp-users": {"analyst"}},
	}

	decision := engine.Decide(Request{
		Actor: Actor{Subject: "user@example.com", Groups: []string{"mcp-users"}},
	})
	if decision.Allowed {
		t.Fatalf("decision allowed empty tool: %#v", decision)
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDoctorExampleConfig(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(wd, "..", "..", "configs", "example.yaml")
	if err := runDoctor([]string{"-config", configPath}); err != nil {
		t.Fatalf("runDoctor error: %v", err)
	}
}

func TestRunDoctorMCPToolInvocationFlagsSkipWithoutTokens(t *testing.T) {
	t.Parallel()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wd, "..", "..")
	configPath := filepath.Join(root, "configs", "example.yaml")
	toolCallPath := filepath.Join(root, "docs", "examples", "postgres_select_tools_call.json")
	if err := runDoctor([]string{
		"-config", configPath,
		"-mcp-url", "https://gateway.example.com/mcp",
		"-tool-call-file", toolCallPath,
	}); err != nil {
		t.Fatalf("runDoctor error: %v", err)
	}
}

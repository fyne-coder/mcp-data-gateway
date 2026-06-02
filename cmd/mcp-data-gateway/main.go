package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fyne-coder/mcp-data-gateway/internal/buildinfo"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/fyne-coder/mcp-data-gateway/internal/doctor"
	"github.com/fyne-coder/mcp-data-gateway/internal/gateway"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	switch args[0] {
	case "version":
		fmt.Printf("mcp-data-gateway %s commit=%s date=%s\n", buildinfo.Version, buildinfo.Commit, buildinfo.Date)
		return nil
	case "doctor":
		return runDoctor(args[1:])
	case "serve":
		return runServe(args[1:])
	default:
		return usage()
	}
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	configPath := fs.String("config", "configs/example.yaml", "path to YAML config")
	checkJWKS := fs.Bool("check-jwks", false, "fetch configured JWKS URL or run OIDC discovery")
	allowedTokenFile := fs.String("allowed-token-file", "", "path to bearer token that must verify and authorize")
	deniedTokenFile := fs.String("denied-token-file", "", "path to bearer token that must verify but fail group authorization")
	mcpURL := fs.String("mcp-url", "", "live /mcp endpoint URL for optional tool invocation checks")
	toolCallFile := fs.String("tool-call-file", "", "path to JSON-RPC request body for optional tool invocation checks")
	timeout := fs.Duration("timeout", 10*time.Second, "timeout for optional network checks")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	return doctor.Run(ctx, os.Stdout, cfg, doctor.Options{
		CheckJWKS:        *checkJWKS,
		AllowedTokenFile: *allowedTokenFile,
		DeniedTokenFile:  *deniedTokenFile,
		MCPURL:           *mcpURL,
		ToolCallFile:     *toolCallFile,
		HTTPClient:       &http.Client{Timeout: *timeout},
	})
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	path := fs.String("config", "configs/example.yaml", "path to YAML config")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	server, err := gateway.New(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("starting gateway: %s\n", server.Summary())
	return server.ListenAndServe()
}

func usage() error {
	return fmt.Errorf("usage: mcp-data-gateway <version|doctor|serve> [flags]")
}

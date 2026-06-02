package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fyne-coder/mcp-data-gateway/internal/audit"
	"github.com/fyne-coder/mcp-data-gateway/internal/auth"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
)

// Options configures doctor checks. HTTPClient and VerifierFactory are optional
// hooks for deterministic tests.
type Options struct {
	CheckJWKS        bool
	AllowedTokenFile string
	DeniedTokenFile  string
	MCPURL           string
	ToolCallFile     string
	HTTPClient       *http.Client
	VerifierFactory  func(context.Context, config.AuthConfig) (auth.Verifier, error)
}

// Run executes doctor checks and writes stable PASS/SKIP/FAIL lines to w.
// It returns a non-nil error when any check fails.
func Run(ctx context.Context, w io.Writer, cfg config.Config, opts Options) error {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	verifierFactory := opts.VerifierFactory
	if verifierFactory == nil {
		verifierFactory = defaultVerifierFactory
	}

	var failed bool
	emit := func(line string) {
		fmt.Fprintln(w, line)
	}

	emit("PASS config")

	if _, err := audit.NewSink(cfg.Audit); err != nil {
		emit(fmt.Sprintf("FAIL audit sink: %v", err))
		failed = true
	} else {
		emit("PASS audit sink")
	}

	if !opts.CheckJWKS {
		emit("SKIP jwks (use --check-jwks)")
	} else if err := checkJWKS(ctx, client, cfg.Auth); err != nil {
		emit(fmt.Sprintf("FAIL jwks: %v", err))
		failed = true
	} else {
		emit("PASS jwks")
	}

	if opts.AllowedTokenFile == "" {
		emit("SKIP allowed token (use --allowed-token-file)")
	} else if err := checkAllowedToken(ctx, cfg, opts.AllowedTokenFile, verifierFactory); err != nil {
		emit(fmt.Sprintf("FAIL allowed token: %v", err))
		failed = true
	} else {
		emit("PASS allowed token")
	}

	if opts.DeniedTokenFile == "" {
		emit("SKIP denied token (use --denied-token-file)")
	} else if err := checkDeniedToken(ctx, cfg, opts.DeniedTokenFile, verifierFactory); err != nil {
		emit(fmt.Sprintf("FAIL denied token: %v", err))
		failed = true
	} else {
		emit("PASS denied token")
	}

	if opts.MCPURL == "" || opts.ToolCallFile == "" || opts.AllowedTokenFile == "" {
		emit("SKIP allowed tool invocation (use --mcp-url, --tool-call-file, and --allowed-token-file)")
	} else if err := checkAllowedToolInvocation(ctx, client, opts.MCPURL, opts.ToolCallFile, opts.AllowedTokenFile); err != nil {
		emit(fmt.Sprintf("FAIL allowed tool invocation: %v", err))
		failed = true
	} else {
		emit("PASS allowed tool invocation")
	}

	if opts.MCPURL == "" || opts.ToolCallFile == "" || opts.DeniedTokenFile == "" {
		emit("SKIP denied tool invocation (use --mcp-url, --tool-call-file, and --denied-token-file)")
	} else if err := checkDeniedToolInvocation(ctx, client, opts.MCPURL, opts.ToolCallFile, opts.DeniedTokenFile); err != nil {
		emit(fmt.Sprintf("FAIL denied tool invocation: %v", err))
		failed = true
	} else {
		emit("PASS denied tool invocation")
	}

	if failed {
		return errors.New("doctor checks failed")
	}
	return nil
}

func defaultVerifierFactory(ctx context.Context, authCfg config.AuthConfig) (auth.Verifier, error) {
	return auth.NewOIDCVerifier(ctx, authCfg)
}

func checkJWKS(ctx context.Context, client *http.Client, authCfg config.AuthConfig) error {
	if authCfg.JWKSURL != "" {
		return fetchJWKS(ctx, client, authCfg.JWKSURL)
	}
	if authCfg.Issuer == "" {
		return fmt.Errorf("auth.issuer is required for JWKS discovery")
	}
	_, err := auth.NewOIDCVerifier(ctx, authCfg)
	return err
}

func fetchJWKS(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}
	var body struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return fmt.Errorf("jwks endpoint did not return JSON")
	}
	if len(body.Keys) == 0 {
		return fmt.Errorf("jwks endpoint did not return keys")
	}
	return nil
}

func checkAllowedToken(
	ctx context.Context,
	cfg config.Config,
	path string,
	verifierFactory func(context.Context, config.AuthConfig) (auth.Verifier, error),
) error {
	token, err := readTokenFile(path)
	if err != nil {
		return err
	}
	verifier, err := verifierFactory(ctx, cfg.Auth)
	if err != nil {
		return err
	}
	actor, err := verifier.Verify(ctx, token)
	if err != nil {
		return fmt.Errorf("verify failed")
	}
	if err := auth.Authorize(actor, cfg.Auth.RequiredGroups); err != nil {
		return fmt.Errorf("authorize failed")
	}
	return nil
}

func checkDeniedToken(
	ctx context.Context,
	cfg config.Config,
	path string,
	verifierFactory func(context.Context, config.AuthConfig) (auth.Verifier, error),
) error {
	token, err := readTokenFile(path)
	if err != nil {
		return err
	}
	verifier, err := verifierFactory(ctx, cfg.Auth)
	if err != nil {
		return err
	}
	actor, err := verifier.Verify(ctx, token)
	if err != nil {
		return fmt.Errorf("verify failed")
	}
	authErr := auth.Authorize(actor, cfg.Auth.RequiredGroups)
	if authErr == nil {
		return fmt.Errorf("token authorized but required group denial")
	}
	if !errors.Is(authErr, auth.ErrForbidden) {
		return fmt.Errorf("authorize returned unexpected error")
	}
	return nil
}

func readTokenFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file is empty")
	}
	return token, nil
}

func readToolCallFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	body := bytes.TrimSpace(data)
	if len(body) == 0 {
		return nil, fmt.Errorf("tool call file is empty")
	}
	return body, nil
}

func postMCPToolCall(ctx context.Context, client *http.Client, mcpURL, token string, requestBody []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mcpURL, bytes.NewReader(requestBody))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

func checkAllowedToolInvocation(ctx context.Context, client *http.Client, mcpURL, toolCallFile, allowedTokenFile string) error {
	token, err := readTokenFile(allowedTokenFile)
	if err != nil {
		return err
	}
	requestBody, err := readToolCallFile(toolCallFile)
	if err != nil {
		return err
	}
	status, body, err := postMCPToolCall(ctx, client, mcpURL, token, requestBody)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("mcp endpoint returned status %d", status)
	}
	if audit.MCPResultStatus(status, body) != audit.ResultOK {
		return fmt.Errorf("protocol-level error")
	}
	return nil
}

func checkDeniedToolInvocation(ctx context.Context, client *http.Client, mcpURL, toolCallFile, deniedTokenFile string) error {
	token, err := readTokenFile(deniedTokenFile)
	if err != nil {
		return err
	}
	requestBody, err := readToolCallFile(toolCallFile)
	if err != nil {
		return err
	}
	status, body, err := postMCPToolCall(ctx, client, mcpURL, token, requestBody)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return nil
	}
	if status >= 200 && status < 300 && audit.MCPResultStatus(status, body) == audit.ResultOK {
		return fmt.Errorf("denied token received protocol success")
	}
	if status >= 200 && status < 300 {
		return nil
	}
	return fmt.Errorf("mcp endpoint returned status %d", status)
}

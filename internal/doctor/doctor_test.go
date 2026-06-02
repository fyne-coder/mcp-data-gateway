package doctor

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fyne-coder/mcp-data-gateway/internal/auth"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type countingVerifier struct {
	actor auth.Actor
	calls int
}

func (v *countingVerifier) Verify(context.Context, string) (auth.Actor, error) {
	v.calls++
	return v.actor, nil
}

type testKey struct {
	keyID string
	key   *rsa.PrivateKey
}

func newTestKey(t *testing.T) testKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return testKey{keyID: "test-key", key: key}
}

func (k testKey) jwksJSON() []byte {
	n := base64.RawURLEncoding.EncodeToString(k.key.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.key.E)).Bytes())
	body, err := json.Marshal(map[string]any{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"kid": k.keyID,
				"alg": "RS256",
				"use": "sig",
				"n":   n,
				"e":   e,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return body
}

func (k testKey) signToken(t *testing.T, issuer, audience string, claims map[string]any) string {
	t.Helper()
	now := time.Now()
	if _, ok := claims["sub"]; !ok {
		claims["sub"] = "user-oidc"
	}
	if _, ok := claims["iss"]; !ok {
		claims["iss"] = issuer
	}
	if _, ok := claims["aud"]; !ok {
		claims["aud"] = audience
	}
	if _, ok := claims["iat"]; !ok {
		claims["iat"] = now.Unix()
	}
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = now.Add(time.Hour).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
	token.Header["kid"] = k.keyID
	signed, err := token.SignedString(k.key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func startJWKSServer(t *testing.T, key testKey) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(key.jwksJSON())
	}))
}

func validExampleConfig() config.Config {
	return config.Config{
		Server: config.ServerConfig{Listen: "127.0.0.1:8080"},
		Edge:   config.EdgeConfig{Mode: config.EdgePublicIngress},
		Auth: config.AuthConfig{
			Issuer:         "https://idp.test",
			Audience:       "mcp-data-gateway",
			GroupClaimName: "groups",
			RequiredGroups: []string{"mcp-users"},
		},
		Policy: config.PolicyConfig{
			DefaultToolPack: "analyst",
			GroupToolPacks:  map[string][]string{"mcp-users": {"analyst"}},
		},
		Audit: config.AuditConfig{Sink: "stdout"},
	}
}

func writeTokenFile(t *testing.T, token string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunDefaultSkipsNetworkAndTokenChecks(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, validExampleConfig(), Options{})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"PASS config",
		"PASS audit sink",
		"SKIP jwks",
		"SKIP allowed token",
		"SKIP denied token",
		"SKIP allowed tool invocation",
		"SKIP denied tool invocation",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "next checks to implement") {
		t.Fatalf("placeholder output still present:\n%s", out)
	}
}

func TestRunUnsupportedAuditSinkFails(t *testing.T) {
	t.Parallel()

	cfg := validExampleConfig()
	cfg.Audit.Sink = "syslog"

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{})
	if err == nil {
		t.Fatal("expected error for unsupported audit sink")
	}
	if !strings.Contains(buf.String(), "FAIL audit sink") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRunCheckJWKSWithPinnedURL(t *testing.T) {
	t.Parallel()

	key := newTestKey(t)
	jwks := startJWKSServer(t, key)
	t.Cleanup(jwks.Close)

	cfg := validExampleConfig()
	cfg.Auth.JWKSURL = jwks.URL

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{
		CheckJWKS:  true,
		HTTPClient: jwks.Client(),
	})
	if err != nil {
		t.Fatalf("Run error: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "PASS jwks") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRunCheckJWKSRejectsNonJWKSBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>login</html>`))
	}))
	t.Cleanup(server.Close)

	cfg := validExampleConfig()
	cfg.Auth.JWKSURL = server.URL

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{
		CheckJWKS:  true,
		HTTPClient: server.Client(),
	})
	if err == nil {
		t.Fatalf("expected non-JWKS body failure:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "FAIL jwks") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRunCheckJWKSMissingIssuerFailsClearly(t *testing.T) {
	t.Parallel()

	cfg := validExampleConfig()
	cfg.Auth.Issuer = ""
	cfg.Auth.JWKSURL = ""

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{CheckJWKS: true})
	if err == nil {
		t.Fatalf("expected missing issuer failure:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "auth.issuer is required for JWKS discovery") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRunAllowedTokenUsesVerifier(t *testing.T) {
	t.Parallel()

	const issuer = "https://idp.test"
	const audience = "mcp-data-gateway"

	key := newTestKey(t)
	jwks := startJWKSServer(t, key)
	t.Cleanup(jwks.Close)

	token := key.signToken(t, issuer, audience, map[string]any{"groups": []string{"mcp-users"}})
	path := writeTokenFile(t, token)

	cfg := validExampleConfig()
	cfg.Auth.Issuer = issuer
	cfg.Auth.Audience = audience
	cfg.Auth.JWKSURL = jwks.URL

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{
		AllowedTokenFile: path,
		HTTPClient:       jwks.Client(),
	})
	if err != nil {
		t.Fatalf("Run error: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "PASS allowed token") {
		t.Fatalf("output = %q", buf.String())
	}
	if strings.Contains(buf.String(), token) {
		t.Fatal("doctor output leaked token value")
	}
}

func TestRunAllowedTokenCallsVerifier(t *testing.T) {
	t.Parallel()

	token := "opaque-test-token"
	path := writeTokenFile(t, token)
	verifier := &countingVerifier{actor: auth.Actor{Subject: "user", Groups: []string{"mcp-users"}}}

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, validExampleConfig(), Options{
		AllowedTokenFile: path,
		VerifierFactory: func(context.Context, config.AuthConfig) (auth.Verifier, error) {
			return verifier, nil
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v\n%s", err, buf.String())
	}
	if verifier.calls != 1 {
		t.Fatalf("Verify calls = %d, want 1", verifier.calls)
	}
	if strings.Contains(buf.String(), token) {
		t.Fatal("doctor output leaked token value")
	}
}

func TestRunDeniedTokenMissingGroupPasses(t *testing.T) {
	t.Parallel()

	const issuer = "https://idp.test"
	const audience = "mcp-data-gateway"

	key := newTestKey(t)
	jwks := startJWKSServer(t, key)
	t.Cleanup(jwks.Close)

	token := key.signToken(t, issuer, audience, map[string]any{"groups": []string{"other-group"}})
	path := writeTokenFile(t, token)

	cfg := validExampleConfig()
	cfg.Auth.Issuer = issuer
	cfg.Auth.Audience = audience
	cfg.Auth.JWKSURL = jwks.URL

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{
		DeniedTokenFile: path,
		HTTPClient:      jwks.Client(),
	})
	if err != nil {
		t.Fatalf("Run error: %v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "PASS denied token") {
		t.Fatalf("output = %q", buf.String())
	}
	if strings.Contains(buf.String(), token) {
		t.Fatal("doctor output leaked token value")
	}
}

func TestRunDeniedTokenWithRequiredGroupFails(t *testing.T) {
	t.Parallel()

	const issuer = "https://idp.test"
	const audience = "mcp-data-gateway"

	key := newTestKey(t)
	jwks := startJWKSServer(t, key)
	t.Cleanup(jwks.Close)

	token := key.signToken(t, issuer, audience, map[string]any{"groups": []string{"mcp-users"}})
	path := writeTokenFile(t, token)

	cfg := validExampleConfig()
	cfg.Auth.Issuer = issuer
	cfg.Auth.Audience = audience
	cfg.Auth.JWKSURL = jwks.URL

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{DeniedTokenFile: path})
	if err == nil {
		t.Fatalf("expected failure when denied token authorizes:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "FAIL denied token") {
		t.Fatalf("output = %q", buf.String())
	}
	if strings.Contains(buf.String(), token) {
		t.Fatal("doctor output leaked token value")
	}
}

func TestRunAllowedTokenFailsWhenVerifierBypassed(t *testing.T) {
	t.Parallel()

	token := "bypass-token"
	path := writeTokenFile(t, token)

	cfg := validExampleConfig()
	bypass := auth.FakeVerifier{Tokens: map[string]auth.Actor{
		token: {Subject: "user", Groups: []string{"mcp-users"}},
	}}

	var buf bytes.Buffer
	err := Run(context.Background(), &buf, cfg, Options{
		AllowedTokenFile: path,
		VerifierFactory: func(context.Context, config.AuthConfig) (auth.Verifier, error) {
			return bypass, nil
		},
	})
	if err != nil {
		t.Fatalf("Run with fake verifier should pass allowed check: %v", err)
	}

	// Real verifier rejects unknown token; proves doctor uses injected factory path
	// that maps to production NewOIDCVerifier by default.
	realBuf := bytes.Buffer{}
	err = Run(context.Background(), &realBuf, cfg, Options{
		AllowedTokenFile: path,
		HTTPClient:       &http.Client{Timeout: time.Second},
	})
	if err == nil {
		t.Fatalf("expected failure with real verifier for unsigned token:\n%s", realBuf.String())
	}
}

const exampleToolsCallBody = `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"postgres_select","arguments":{"table":"public.example_table","columns":["id","name"],"limit":10}}}`

func writeToolCallFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool-call.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func startMCPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestRunAllowedToolInvocationPassesOnProtocolSuccess(t *testing.T) {
	t.Parallel()

	const allowedToken = "allowed-live-token"
	mcp := startMCPServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+allowedToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":5,"result":{"content":[{"type":"text","text":"ok"}]}}`))
	})
	t.Cleanup(mcp.Close)

	toolCall := writeToolCallFile(t, exampleToolsCallBody)
	allowedPath := writeTokenFile(t, allowedToken)

	verifier := &countingVerifier{actor: auth.Actor{Subject: "user", Groups: []string{"mcp-users"}}}
	var buf bytes.Buffer
	err := Run(context.Background(), &buf, validExampleConfig(), Options{
		MCPURL:           mcp.URL,
		ToolCallFile:     toolCall,
		AllowedTokenFile: allowedPath,
		HTTPClient:       mcp.Client(),
		VerifierFactory: func(context.Context, config.AuthConfig) (auth.Verifier, error) {
			return verifier, nil
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v\n%s", err, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "PASS allowed tool invocation") {
		t.Fatalf("output = %q", out)
	}
	if strings.Contains(out, allowedToken) || strings.Contains(out, exampleToolsCallBody) {
		t.Fatal("doctor output leaked token or request body")
	}
}

func TestRunAllowedToolInvocationFailsOnJSONRPCError(t *testing.T) {
	t.Parallel()

	const allowedToken = "allowed-live-token"
	mcp := startMCPServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":5,"error":{"code":-32603,"message":"tool failed"}}`))
	})
	t.Cleanup(mcp.Close)

	toolCall := writeToolCallFile(t, exampleToolsCallBody)
	allowedPath := writeTokenFile(t, allowedToken)

	verifier := &countingVerifier{actor: auth.Actor{Subject: "user", Groups: []string{"mcp-users"}}}
	var buf bytes.Buffer
	err := Run(context.Background(), &buf, validExampleConfig(), Options{
		MCPURL:           mcp.URL,
		ToolCallFile:     toolCall,
		AllowedTokenFile: allowedPath,
		HTTPClient:       mcp.Client(),
		VerifierFactory: func(context.Context, config.AuthConfig) (auth.Verifier, error) {
			return verifier, nil
		},
	})
	if err == nil {
		t.Fatalf("expected failure:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "FAIL allowed tool invocation") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestRunDeniedToolInvocationPassesOnHTTP403(t *testing.T) {
	t.Parallel()

	const deniedToken = "denied-live-token"
	mcp := startMCPServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	t.Cleanup(mcp.Close)

	toolCall := writeToolCallFile(t, exampleToolsCallBody)
	deniedPath := writeTokenFile(t, deniedToken)

	verifier := &countingVerifier{actor: auth.Actor{Subject: "user", Groups: []string{"other-group"}}}
	var buf bytes.Buffer
	err := Run(context.Background(), &buf, validExampleConfig(), Options{
		MCPURL:          mcp.URL,
		ToolCallFile:    toolCall,
		DeniedTokenFile: deniedPath,
		HTTPClient:      mcp.Client(),
		VerifierFactory: func(context.Context, config.AuthConfig) (auth.Verifier, error) {
			return verifier, nil
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v\n%s", err, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "PASS denied tool invocation") {
		t.Fatalf("output = %q", out)
	}
	if strings.Contains(out, deniedToken) || strings.Contains(out, exampleToolsCallBody) {
		t.Fatal("doctor output leaked token or request body")
	}
}

func TestRunDeniedToolInvocationFailsOnProtocolSuccess(t *testing.T) {
	t.Parallel()

	const deniedToken = "denied-live-token"
	mcp := startMCPServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":5,"result":{"content":[{"type":"text","text":"ok"}]}}`))
	})
	t.Cleanup(mcp.Close)

	toolCall := writeToolCallFile(t, exampleToolsCallBody)
	deniedPath := writeTokenFile(t, deniedToken)

	verifier := &countingVerifier{actor: auth.Actor{Subject: "user", Groups: []string{"other-group"}}}
	var buf bytes.Buffer
	err := Run(context.Background(), &buf, validExampleConfig(), Options{
		MCPURL:          mcp.URL,
		ToolCallFile:    toolCall,
		DeniedTokenFile: deniedPath,
		HTTPClient:      mcp.Client(),
		VerifierFactory: func(context.Context, config.AuthConfig) (auth.Verifier, error) {
			return verifier, nil
		},
	})
	if err == nil {
		t.Fatalf("expected failure:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "FAIL denied tool invocation") {
		t.Fatalf("output = %q", buf.String())
	}
}

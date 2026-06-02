package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

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

func signHMACToken(t *testing.T, issuer, audience string, claims map[string]any) string {
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	signed, err := token.SignedString([]byte("shared-secret"))
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

func TestOIDCVerifierWithPinnedJWKS(t *testing.T) {
	t.Parallel()

	const issuer = "https://idp.test"
	const audience = "mcp-data-gateway"

	key := newTestKey(t)
	jwks := startJWKSServer(t, key)
	t.Cleanup(jwks.Close)

	verifier, err := NewOIDCVerifier(context.Background(), config.AuthConfig{
		Issuer:         issuer,
		Audience:       audience,
		JWKSURL:        jwks.URL,
		GroupClaimName: "groups",
	})
	if err != nil {
		t.Fatalf("NewOIDCVerifier error: %v", err)
	}

	valid := key.signToken(t, issuer, audience, map[string]any{"groups": []string{"mcp-users"}})
	actor, err := verifier.Verify(context.Background(), valid)
	if err != nil {
		t.Fatalf("Verify valid token error: %v", err)
	}
	if actor.Subject != "user-oidc" {
		t.Fatalf("subject = %q, want user-oidc", actor.Subject)
	}

	invalid := key.signToken(t, issuer, "wrong-audience", map[string]any{"groups": []string{"mcp-users"}})
	if _, err := verifier.Verify(context.Background(), invalid); err != ErrUnauthorized {
		t.Fatalf("Verify wrong audience = %v, want ErrUnauthorized", err)
	}

	wrongIssuer := key.signToken(t, issuer, audience, map[string]any{
		"iss":    "https://wrong-idp.test",
		"groups": []string{"mcp-users"},
	})
	if _, err := verifier.Verify(context.Background(), wrongIssuer); err != ErrUnauthorized {
		t.Fatalf("Verify wrong issuer = %v, want ErrUnauthorized", err)
	}

	expired := key.signToken(t, issuer, audience, map[string]any{
		"exp":    time.Now().Add(-time.Hour).Unix(),
		"groups": []string{"mcp-users"},
	})
	if _, err := verifier.Verify(context.Background(), expired); err != ErrUnauthorized {
		t.Fatalf("Verify expired token = %v, want ErrUnauthorized", err)
	}

	missingSubject := key.signToken(t, issuer, audience, map[string]any{
		"sub":    "",
		"groups": []string{"mcp-users"},
	})
	if _, err := verifier.Verify(context.Background(), missingSubject); err != ErrUnauthorized {
		t.Fatalf("Verify missing subject = %v, want ErrUnauthorized", err)
	}

	hs256 := signHMACToken(t, issuer, audience, map[string]any{"groups": []string{"mcp-users"}})
	if _, err := verifier.Verify(context.Background(), hs256); err != ErrUnauthorized {
		t.Fatalf("Verify HS256 token = %v, want ErrUnauthorized", err)
	}
}

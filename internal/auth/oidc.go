package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fyne-coder/mcp-data-gateway/internal/config"
)

// OIDCVerifier validates JWT access tokens with go-oidc/JWKS. go-oidc exposes
// IDTokenVerifier as its maintained verifier for issuer, audience, expiry, and
// signature checks; this gateway requires JWT access tokens with those claims.
type OIDCVerifier struct {
	verifier   *oidc.IDTokenVerifier
	groupClaim string
}

// NewOIDCVerifier builds a verifier from auth config. When JWKSURL is set, discovery
// is skipped so tests and offline deployments can pin keys deterministically.
func NewOIDCVerifier(ctx context.Context, auth config.AuthConfig) (*OIDCVerifier, error) {
	oidcCfg := &oidc.Config{ClientID: auth.Audience}

	var idVerifier *oidc.IDTokenVerifier
	if auth.JWKSURL != "" {
		keySet := oidc.NewRemoteKeySet(ctx, auth.JWKSURL)
		idVerifier = oidc.NewVerifier(auth.Issuer, keySet, oidcCfg)
	} else {
		provider, err := oidc.NewProvider(ctx, auth.Issuer)
		if err != nil {
			return nil, fmt.Errorf("oidc provider: %w", err)
		}
		idVerifier = provider.Verifier(oidcCfg)
	}

	groupClaim := auth.GroupClaimName
	if groupClaim == "" {
		groupClaim = "groups"
	}

	return &OIDCVerifier{verifier: idVerifier, groupClaim: groupClaim}, nil
}

func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (Actor, error) {
	idToken, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return Actor{}, ErrUnauthorized
	}

	var payload map[string]json.RawMessage
	if err := idToken.Claims(&payload); err != nil {
		return Actor{}, ErrUnauthorized
	}

	subject := idToken.Subject
	if subject == "" {
		return Actor{}, ErrUnauthorized
	}

	groups, err := parseGroupsClaim(payload[v.groupClaim])
	if err != nil {
		return Actor{}, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	return Actor{Subject: subject, Groups: groups}, nil
}

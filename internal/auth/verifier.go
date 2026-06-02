package auth

import "context"

// Verifier validates bearer tokens and returns actor claims.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (Actor, error)
}

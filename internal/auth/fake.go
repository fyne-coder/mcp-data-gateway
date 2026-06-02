package auth

import "context"

// FakeVerifier maps raw tokens to actors for deterministic gateway tests.
type FakeVerifier struct {
	Tokens map[string]Actor
	Err    error
}

func (f FakeVerifier) Verify(_ context.Context, rawToken string) (Actor, error) {
	if f.Err != nil {
		return Actor{}, f.Err
	}
	actor, ok := f.Tokens[rawToken]
	if !ok {
		return Actor{}, ErrUnauthorized
	}
	return actor, nil
}

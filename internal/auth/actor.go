package auth

// Actor holds identity claims extracted from a verified bearer token.
type Actor struct {
	Subject string
	Groups  []string
}

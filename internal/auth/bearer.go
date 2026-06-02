package auth

import (
	"strings"
)

// ParseBearer extracts the bearer token from an Authorization header value.
func ParseBearer(header string) (string, error) {
	if header == "" {
		return "", ErrUnauthorized
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", ErrUnauthorized
	}
	return strings.TrimSpace(parts[1]), nil
}

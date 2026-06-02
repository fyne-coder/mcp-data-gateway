package audit

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const requestIDHeader = "X-Request-Id"

// RequestID returns the inbound request ID or generates a non-empty value.
func RequestID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get(requestIDHeader)); id != "" {
		return id
	}
	return newID("req")
}

// NewEventID returns a unique event identifier.
func NewEventID() string {
	return newID("evt")
}

func newID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_fallback"
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

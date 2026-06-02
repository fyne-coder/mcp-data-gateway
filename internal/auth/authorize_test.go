package auth

import "testing"

func TestAuthorize(t *testing.T) {
	t.Parallel()

	actor := Actor{Subject: "user-1", Groups: []string{"mcp-users", "other"}}
	if err := Authorize(actor, []string{"mcp-users"}); err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if err := Authorize(actor, []string{"admins"}); err != ErrForbidden {
		t.Fatalf("Authorize = %v, want ErrForbidden", err)
	}
}

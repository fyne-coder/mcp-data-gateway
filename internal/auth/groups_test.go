package auth

import (
	"encoding/json"
	"testing"
)

func TestParseGroupsClaim(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "array", raw: `["a","b"]`, want: []string{"a", "b"}},
		{name: "string", raw: `"solo"`, want: []string{"solo"}},
		{name: "empty", raw: ``, want: nil},
		{name: "invalid type", raw: `123`, wantErr: true},
		{name: "mixed array", raw: `[1,"a"]`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseGroupsClaim(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseGroupsClaim succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGroupsClaim error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("groups = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("groups[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

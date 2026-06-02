package auth

import "testing"

func TestParseBearer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{name: "valid", header: "Bearer token-abc", want: "token-abc"},
		{name: "missing", header: "", wantErr: true},
		{name: "not bearer", header: "Basic abc", wantErr: true},
		{name: "empty token", header: "Bearer ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseBearer(tt.header)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ParseBearer succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBearer error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("token = %q, want %q", got, tt.want)
			}
		})
	}
}

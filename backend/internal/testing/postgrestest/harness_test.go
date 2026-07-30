package postgrestest

import (
	"strings"
	"testing"
)

func TestSafeServerVersion(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "debian build", raw: "16.14 (Debian 16.14-1.pgdg13+1)", want: "16.14"},
		{name: "dotted version", raw: "17.9", want: "17.9"},
		{name: "three-part version", raw: "9.6.24", want: "9.6.24"},
		{name: "major version", raw: "18", want: "18"},
		{name: "surrounding whitespace", raw: "  15.13 (Ubuntu 15.13-1)  ", want: "15.13"},
		{name: "empty", raw: "", want: "unknown"},
		{name: "malformed", raw: "PostgreSQL 16.14", want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeServerVersion(tt.raw)
			if got != tt.want {
				t.Fatalf("safeServerVersion(%q) = %q, want %q", tt.raw, got, tt.want)
			}
			if strings.ContainsAny(got, " \t\r\n") {
				t.Fatalf("safeServerVersion(%q) returned whitespace: %q", tt.raw, got)
			}
		})
	}
}

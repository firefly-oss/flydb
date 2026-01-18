package sql

import (
	"testing"
)

func TestMatchLikeHelper(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		pattern string
		want    bool
	}{
		{"Exact match", "hello", "hello", true},
		{"Case insensitive match", "Hello", "hello", true},
		{"Case insensitive match 2", "hello", "HELLO", true},
		{"Case insensitive match 3", "HeLLo", "hEllO", true},
		{"Mismatch", "hello", "world", false},
		{"Wildcard % match all", "hello", "%", true},
		{"Wildcard % match prefix", "hello", "h%", true},
		{"Wildcard % match suffix", "hello", "%o", true},
		{"Wildcard % match middle", "hello", "h%o", true},
		{"Wildcard _ match single", "hello", "hell_", true},
		{"Wildcard _ match single 2", "hello", "_ello", true},
		{"Mixed wildcards", "hello", "h_ll%", true},
		{"Empty string", "", "", true},
		{"Empty string match %", "", "%", true},
		{"Empty string mismatch _", "", "_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchLikePattern(tt.str, tt.pattern); got != tt.want {
				t.Errorf("matchLikePattern(%q, %q) = %v, want %v", tt.str, tt.pattern, got, tt.want)
			}
		})
	}
}

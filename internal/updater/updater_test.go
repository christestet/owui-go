package updater

import (
	"testing"
	"time"
)

func TestShouldCheck(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", true},
		{"invalid", "not-a-time", true},
		{"25h ago", time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339), true},
		{"23h ago", time.Now().Add(-23 * time.Hour).UTC().Format(time.RFC3339), false},
		{"just now", time.Now().UTC().Format(time.RFC3339), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldCheck(tt.in); got != tt.want {
				t.Errorf("ShouldCheck(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

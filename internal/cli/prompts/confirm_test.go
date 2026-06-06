package prompts

import (
	"strings"
	"testing"
)

func TestConfirmYN(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"lowercase y", "y\n", true},
		{"uppercase Y", "Y\n", true},
		{"y with spaces", "  y  \n", true},
		{"explicit no", "n\n", false},
		{"empty/enter defaults to no", "\n", false},
		{"yes word is not y", "yes\n", false},
		{"eof defaults to no", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			got, err := ConfirmYN(strings.NewReader(tt.input), &out, "Proceed?")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ConfirmYN(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if !strings.Contains(out.String(), "Proceed? [y/N]: ") {
				t.Errorf("expected prompt written to out, got %q", out.String())
			}
		})
	}
}

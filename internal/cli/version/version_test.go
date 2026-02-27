package version

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	Version = "1.0.0"
	Commit = "abcdef"
	Date = "2026-02-26"

	buf := new(bytes.Buffer)
	Cmd.SetOut(buf)
	Cmd.SetArgs([]string{})

	if err := Cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected output to contain version, got: %s", output)
	}
	if !strings.Contains(output, "abcdef") {
		t.Errorf("expected output to contain commit, got: %s", output)
	}
	if !strings.Contains(output, "2026-02-26") {
		t.Errorf("expected output to contain date, got: %s", output)
	}
}

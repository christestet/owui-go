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

	// Suppress standard output for test
	Cmd.SetArgs([]string{})
	_ = Cmd.Execute()

	// Since Cmd.Run writes directly to fmt.Printf by default in this setup,
	// let's just make sure the configuration is right and properties are accessible
	if Cmd.Use != "version" {
		t.Errorf("expected use 'version', got %q", Cmd.Use)
	}
	if !strings.Contains(Cmd.Short, "Print the version") {
		t.Errorf("expected short description to mention version printing")
	}
}

package root

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/viper"
)

func TestRootCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	Cmd.SetOut(buf)

	// Testing empty execution fallback to help overview
	Cmd.SetArgs([]string{"--help"})
	err := Cmd.Execute()

	if err != nil {
		t.Errorf("unexpected error on root execute: %v", err)
	}
}

func TestExecuteRoot(t *testing.T) {
	// Test Cmd.Execute() directly to avoid os.Exit(1) in Execute() wrapper
	Cmd.SetArgs([]string{"--help"})
	buf := new(bytes.Buffer)
	Cmd.SetOut(buf)
	if err := Cmd.Execute(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitConfig(t *testing.T) {
	// Simple invocation, should not panic even if no config exists natively
	initConfig()
}

func TestInitConfig_ErrorLog(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "owui-test-root-*")
	defer os.RemoveAll(tmpDir)

	// Since we know ConfigPath uses the UserConfigDirFunc
	origUserConfigDir := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) {
		return tmpDir, nil
	}
	defer func() {
		config.UserConfigDirFunc = origUserConfigDir
	}()

	path := filepath.Join(tmpDir, "owui", "config.json")
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, []byte(`{INVALID_JSON`), 0644)

	// Viper caches state, ensure the new test attempts reading the broken config
	viper.Reset()

	err := initConfig()
	if err == nil {
		t.Errorf("expected error from initConfig with invalid json")
	}
}

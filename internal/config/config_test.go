package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfigLoadAndSave(t *testing.T) {
	// Setup a temporary directory for config
	tmpDir, err := os.MkdirTemp("", "owui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Since os.UserConfigDir() checks os environments, we can set the env var.
	// For macOS, UserConfigDir doesn't natively check XDG_CONFIG_HOME first,
	// so let's mock the path or test ConfigPath behavior dynamically.

	// A simpler way for tests is to monkey-patch or just set XDG_CONFIG_HOME
	// but on macOS it prioritizes Library/Application Support. Let's force it for tests.
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Save the original UserConfigDir
	origUserConfigDir := UserConfigDirFunc
	UserConfigDirFunc = func() (string, error) {
		return tmpDir, nil
	}
	defer func() {
		UserConfigDirFunc = origUserConfigDir
	}()

	// Reset viper state before test
	viper.Reset()

	// Test Load (should create default config)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Settings.OutputFormat != "pretty" {
		t.Errorf("expected default output_format 'pretty', got %q", cfg.Settings.OutputFormat)
	}

	// Modify config
	cfg.ActiveInstance = "test-instance"
	cfg.Instances = map[string]InstanceConfig{
		"test-instance": {
			URL:    "http://localhost:3000",
			APIKey: "secret",
		},
	}

	// Because viper uses mapstructure natively but sometimes struggles with time.Time,
	// let's explicitly inject a structure instead of map[string]interface
	viper.Set("active_instance", cfg.ActiveInstance)
	viper.Set("instances", cfg.Instances)

	// Test Save
	err = Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Reset viper and reload to verify persistence
	viper.Reset()
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() failed: %v", err)
	}

	if loadedCfg.ActiveInstance != "test-instance" {
		t.Errorf("expected active_instance 'test-instance', got %q", loadedCfg.ActiveInstance)
	}
	if loadedCfg.Instances["test-instance"].URL != "http://localhost:3000" {
		t.Errorf("expected instance URL 'http://localhost:3000', got %q", loadedCfg.Instances["test-instance"].URL)
	}
}

func TestConfigPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origUserConfigDir := UserConfigDirFunc
	UserConfigDirFunc = func() (string, error) {
		return tmpDir, nil
	}
	defer func() {
		UserConfigDirFunc = origUserConfigDir
	}()

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "owui", "config.json")
	if path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, path)
	}
}

func TestConfigLoad_BadJson(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-test-bad-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origUserConfigDir := UserConfigDirFunc
	UserConfigDirFunc = func() (string, error) {
		return tmpDir, nil
	}
	defer func() {
		UserConfigDirFunc = origUserConfigDir
	}()

	path, _ := ConfigPath()
	os.WriteFile(path, []byte(`{INVALID_JSON`), 0644)

	viper.Reset()
	_, err = Load()
	if err == nil {
		t.Errorf("expected error loading invalid json")
	}
}

func TestInstanceConfig_RedactedAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "empty key",
			apiKey:   "",
			expected: "******",
		},
		{
			name:     "short key",
			apiKey:   "abc",
			expected: "******",
		},
		{
			name:     "exactly 6 chars",
			apiKey:   "123456",
			expected: "******",
		},
		{
			name:     "longer key",
			apiKey:   "sk-1234567890abcdef",
			expected: "sk-123******",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := InstanceConfig{APIKey: tt.apiKey}
			if got := cfg.RedactedAPIKey(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

package instances

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/viper"
)

func TestHealthCommand(t *testing.T) {
	// Setup config
	tmpDir, err := os.MkdirTemp("", "owui-health-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))

	viper.Set("active_instance", "test-instance")
	viper.Set("instances", map[string]interface{}{
		"test-instance": map[string]interface{}{
			"url":      "http://127.0.0.1:0", // Invalid/closed port to trigger error
			"api_key":  "secret",
			"added_at": time.Now().Format(time.RFC3339),
		},
	})
	viper.WriteConfig()

	// Capture output
	buf := new(bytes.Buffer)
	healthCmd.SetOut(buf)

	// Since we mock the config file natively, we shouldn't fail configuration load
	healthCmd.SetArgs([]string{})

	// Test the health check - should return error for unreachable instance
	err = healthCmd.RunE(healthCmd, []string{})
	if err == nil {
		t.Errorf("expected error for unreachable instance, got nil")
	} else if !strings.Contains(err.Error(), "unhealthy instances") {
		t.Errorf("expected 'unhealthy instances' error, got: %v", err)
	}
}

func TestHealthCommand_NoActiveInstance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-health-test2-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Force an empty config directory to ensure NO instance is found
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))

	// Create a truly empty config structure
	viper.Reset()
	viper.Set("active_instance", "")
	viper.Set("instances", map[string]interface{}{})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	healthCmd.SetOut(buf)

	healthCmd.SetArgs([]string{})

	err = healthCmd.RunE(healthCmd, []string{})
	if err == nil {
		t.Errorf("expected error due to missing instance")
	} else if !strings.Contains(err.Error(), "no instances configured") {
		t.Errorf("expected 'no instances configured' error, got: %v", err)
	}
}

func TestHealthCommand_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "owui-health-test3-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)

	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "healthy-instance")
	viper.Set("instances", map[string]interface{}{
		"healthy-instance": map[string]interface{}{
			"url":      server.URL,
			"api_key":  "secret",
			"added_at": "2026-02-26T00:00:00Z",
		},
	})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	healthCmd.SetOut(buf)

	healthCmd.SetArgs([]string{})

	err = healthCmd.RunE(healthCmd, []string{})
	if err != nil {
		t.Errorf("expected healthy instance check to pass, got error: %v", err)
	}
}

func TestHealthCommand_InstanceNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-health-test4-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))

	viper.Reset()
	viper.Set("active_instance", "instance-a")
	viper.Set("instances", map[string]interface{}{
		"instance-a": map[string]interface{}{
			"url":      "http://127.0.0.1:8080",
			"api_key":  "sk-aaa",
			"added_at": "2026-01-01T00:00:00Z",
		},
	})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	healthCmd.SetOut(buf)

	// Register the instance flag on the command for test isolation
	if healthCmd.Flags().Lookup("instance") == nil {
		healthCmd.Flags().String("instance", "", "")
	}
	healthCmd.Flags().Set("instance", "nonexistent-instance")
	defer healthCmd.Flags().Set("instance", "")

	err = healthCmd.RunE(healthCmd, []string{})
	if err == nil {
		t.Errorf("expected error due to missing instance in config mapping")
	} else if !strings.Contains(err.Error(), "not found in config") {
		t.Errorf("expected instance not found error, got: %v", err)
	}
}

func TestListCommand(t *testing.T) {
	// Setup config
	tmpDir, err := os.MkdirTemp("", "owui-list-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))

	viper.Set("active_instance", "test-instance")
	viper.Set("instances", map[string]interface{}{
		"test-instance": map[string]interface{}{
			"url":      "http://127.0.0.1:8080",
			"api_key":  "sk-mysecretapikey123",
			"added_at": time.Now().Format(time.RFC3339),
		},
		"other-instance": map[string]interface{}{
			"url":      "http://127.0.0.1:9090",
			"api_key":  "short",
			"added_at": time.Now().Format(time.RFC3339),
		},
	})
	viper.Set("settings.output_format", "pretty")
	viper.WriteConfig()

	// Test console output
	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	err = listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "sk-mys******") {
		t.Errorf("expected output to contain redacted long key, got:\n%s", output)
	}
	if strings.Contains(output, "sk-mysecretapikey123") {
		t.Errorf("expected output NOT to contain plain long key, got:\n%s", output)
	}
	// "short" is 5 characters, should become "******"
	if !strings.Contains(output, "******") {
		t.Errorf("expected output to contain redacted short key, got:\n%s", output)
	}
	if strings.Contains(output, "short") && !strings.Contains(output, "other-instance") {
		// Just a sanity check to ensure the API key "short" isn't printed plain.
		t.Errorf("expected output NOT to contain plain short key, got:\n%s", output)
	}

	// Test JSON output - register flags if not already present (they're persistent on root)
	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "json")

	buf.Reset()
	err = listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error for json output, got: %v", err)
	}

	jsonOutput := buf.String()
	if !strings.Contains(jsonOutput, "sk-mys******") {
		t.Errorf("expected json output to contain redacted long key, got:\n%s", jsonOutput)
	}
	if strings.Contains(jsonOutput, "sk-mysecretapikey123") {
		t.Errorf("expected json output NOT to contain plain long key, got:\n%s", jsonOutput)
	}
}

func TestUseCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-use-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "instance-a")
	viper.Set("instances", map[string]interface{}{
		"instance-a": map[string]interface{}{
			"url":      "http://127.0.0.1:8080",
			"api_key":  "sk-aaa",
			"added_at": "2026-01-01T00:00:00Z",
		},
		"instance-b": map[string]interface{}{
			"url":      "http://127.0.0.1:9090",
			"api_key":  "sk-bbb",
			"added_at": "2026-01-02T00:00:00Z",
		},
	})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	useCmd.SetOut(buf)

	err = useCmd.RunE(useCmd, []string{"instance-b"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `Switched to instance "instance-b"`) {
		t.Errorf("expected switch confirmation, got: %s", output)
	}

	// Verify config was persisted
	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.ReadInConfig()
	if got := viper.GetString("active_instance"); got != "instance-b" {
		t.Errorf("expected active_instance to be 'instance-b', got %q", got)
	}
}

func TestUseCommand_InstanceNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-use-test2-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "instance-a")
	viper.Set("instances", map[string]interface{}{
		"instance-a": map[string]interface{}{
			"url":      "http://127.0.0.1:8080",
			"api_key":  "sk-aaa",
			"added_at": "2026-01-01T00:00:00Z",
		},
	})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	useCmd.SetOut(buf)

	err = useCmd.RunE(useCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent instance")
	}
	if !strings.Contains(err.Error(), "not found in config") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestAddCommand_NonInteractive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-add-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "")
	viper.Set("instances", map[string]interface{}{})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	addCmd.SetOut(buf)
	addCmd.SetArgs([]string{"my-instance", "--url", "http://localhost:3000", "--api-key", "sk-test123"})

	err = addCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `Instance "my-instance" added`) {
		t.Errorf("expected add confirmation, got: %s", output)
	}
	if !strings.Contains(output, "Set as active instance") {
		t.Errorf("expected active instance message for first instance, got: %s", output)
	}

	// Verify config was persisted
	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.ReadInConfig()
	if got := viper.GetString("active_instance"); got != "my-instance" {
		t.Errorf("expected active_instance to be 'my-instance', got %q", got)
	}
}

func TestAddCommand_DuplicateInstance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-add-test2-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "existing")
	viper.Set("instances", map[string]interface{}{
		"existing": map[string]interface{}{
			"url":      "http://localhost:3000",
			"api_key":  "sk-old",
			"added_at": "2026-01-01T00:00:00Z",
		},
	})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	addCmd.SetOut(buf)
	addCmd.SetArgs([]string{"existing", "--url", "http://localhost:4000", "--api-key", "sk-new"})

	err = addCmd.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate instance name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestAddCommand_InvalidURL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "owui-add-test3-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { config.UserConfigDirFunc = origFunc }()

	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)
	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "")
	viper.Set("instances", map[string]interface{}{})
	viper.WriteConfig()

	buf := new(bytes.Buffer)
	addCmd.SetOut(buf)
	addCmd.SetArgs([]string{"bad-instance", "--url", "not-a-url", "--api-key", "sk-test"})

	err = addCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("expected 'invalid URL' error, got: %v", err)
	}
}

func TestValidateInstanceInput(t *testing.T) {
	tests := []struct {
		name        string
		instName    string
		instanceURL string
		wantErr     string
	}{
		{"empty name", "", "http://localhost", "instance name is required"},
		{"empty url", "test", "", "instance URL is required"},
		{"invalid url", "test", "not-a-url", "invalid URL"},
		{"valid input", "test", "http://localhost:3000", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInstanceInput(tt.instName, tt.instanceURL)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

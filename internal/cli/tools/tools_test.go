package tools

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/christestet/owui-go/internal/api"
	"github.com/christestet/owui-go/internal/config"
	"github.com/spf13/viper"
)

func setupTestConfig(t *testing.T, serverURL string) (cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "owui-tools-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)

	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "test-instance")
	viper.Set("instances", map[string]interface{}{
		"test-instance": map[string]interface{}{
			"url":      serverURL,
			"api_key":  "secret",
			"added_at": "2026-01-01T00:00:00Z",
		},
	})
	viper.Set("settings.timeout_seconds", 5)
	viper.WriteConfig()

	return func() {
		config.UserConfigDirFunc = origFunc
		os.RemoveAll(tmpDir)
	}
}

func setupEmptyConfig(t *testing.T) (cleanup func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "owui-tools-test-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	origFunc := config.UserConfigDirFunc
	config.UserConfigDirFunc = func() (string, error) { return tmpDir, nil }

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	configPath := filepath.Join(tmpDir, "owui")
	os.MkdirAll(configPath, 0700)

	viper.Reset()
	viper.SetConfigFile(filepath.Join(configPath, "config.json"))
	viper.Set("active_instance", "")
	viper.Set("instances", map[string]interface{}{})
	viper.WriteConfig()

	return func() {
		config.UserConfigDirFunc = origFunc
		os.RemoveAll(tmpDir)
	}
}

func newToolsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/tools/list" {
			json.NewEncoder(w).Encode([]api.Tool{
				{
					ID: "weather", UserID: "u1", Name: "Weather",
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
					},
					UpdatedAt: 1700000000,
				},
				{
					ID: "calc", UserID: "u1", Name: "Calculator",
					AccessGrants: []api.AccessGrantModel{},
					UpdatedAt:    1700000100,
				},
				{
					ID: "internal", UserID: "u1", Name: "Internal API",
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g2", Permission: "write"},
					},
					UpdatedAt: 1700000200,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"not found"}`))
	}))
}

func resetListFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"filter", "output"} {
		if listCmd.Flags().Lookup(name) == nil {
			listCmd.Flags().String(name, "", "")
		}
	}
	for _, name := range []string{"filter", "output"} {
		listCmd.Flags().Set(name, "")
	}
}

func TestListCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	err := listCmd.RunE(listCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestListCommand_PrettyOutput(t *testing.T) {
	server := newToolsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	for _, header := range []string{"NAME", "VISIBILITY", "GRANTS", "UPDATED"} {
		if !strings.Contains(output, header) {
			t.Errorf("expected header %q in output, got:\n%s", header, output)
		}
	}
	if !strings.Contains(output, "Weather") || !strings.Contains(output, "private") {
		t.Errorf("expected Weather (private) row, got:\n%s", output)
	}
	if !strings.Contains(output, "Calculator") || !strings.Contains(output, "public") {
		t.Errorf("expected Calculator (public) row, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 3 tool(s).") {
		t.Errorf("expected summary line, got:\n%s", output)
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	server := newToolsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	listCmd.Flags().Set("output", "json")
	defer listCmd.Flags().Set("output", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var tools []api.Tool
	if err := json.Unmarshal(buf.Bytes(), &tools); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\nOutput:\n%s", err, buf.String())
	}
	if len(tools) != 3 {
		t.Errorf("expected 3 tools in JSON, got %d", len(tools))
	}
}

func TestListCommand_FilterPrivate(t *testing.T) {
	server := newToolsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	listCmd.Flags().Set("filter", "private")
	defer listCmd.Flags().Set("filter", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if strings.Contains(output, "Calculator") {
		t.Errorf("public tool leaked into private filter:\n%s", output)
	}
	if !strings.Contains(output, "Showing 2 tool(s) matching filter \"private\".") {
		t.Errorf("expected 2 private tools, got:\n%s", output)
	}
}

func TestListCommand_FilterPublic(t *testing.T) {
	server := newToolsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	listCmd.Flags().Set("filter", "public")
	defer listCmd.Flags().Set("filter", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Calculator") {
		t.Errorf("expected Calculator in public filter, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 1 tool(s) matching filter \"public\".") {
		t.Errorf("expected exactly 1 public tool, got:\n%s", output)
	}
}

func TestListCommand_InvalidFilter(t *testing.T) {
	server := newToolsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	listCmd.Flags().Set("filter", "bogus")
	defer listCmd.Flags().Set("filter", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	err := listCmd.RunE(listCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid filter")
	}
	if !strings.Contains(err.Error(), "invalid filter") {
		t.Errorf("expected 'invalid filter' error, got: %v", err)
	}
}

func TestToolVisibility(t *testing.T) {
	priv := api.Tool{AccessGrants: []api.AccessGrantModel{{PrincipalType: "group", PrincipalID: "g1", Permission: "read"}}}
	pub := api.Tool{AccessGrants: []api.AccessGrantModel{}}
	if toolVisibility(priv) != "private" {
		t.Errorf("expected private, got %q", toolVisibility(priv))
	}
	if toolVisibility(pub) != "public" {
		t.Errorf("expected public, got %q", toolVisibility(pub))
	}
}

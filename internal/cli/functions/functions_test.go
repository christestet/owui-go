package functions

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

	tmpDir, err := os.MkdirTemp("", "owui-functions-test-*")
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

	tmpDir, err := os.MkdirTemp("", "owui-functions-test-empty-*")
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

func newFunctionsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/functions/" {
			json.NewEncoder(w).Encode([]api.Function{
				{ID: "emoji_filter", UserID: "u1", Name: "Emoji Filter", Type: "filter", IsActive: true, IsGlobal: true, UpdatedAt: 1700000000},
				{ID: "tone_filter", UserID: "u1", Name: "Tone Filter", Type: "filter", IsActive: true, IsGlobal: false, UpdatedAt: 1700000100},
				{ID: "draft_action", UserID: "u1", Name: "Draft Action", Type: "action", IsActive: false, IsGlobal: false, UpdatedAt: 1700000200},
				{ID: "private_pipe", UserID: "u2", Name: "Private Pipe", Type: "pipe", IsActive: false, IsGlobal: true, UpdatedAt: 1700000300},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"not found"}`))
	}))
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
	server := newFunctionsServer(t)
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
	for _, header := range []string{"NAME", "TYPE", "STATUS", "SCOPE", "UPDATED"} {
		if !strings.Contains(output, header) {
			t.Errorf("expected header %q in output, got:\n%s", header, output)
		}
	}
	// Global, enabled
	if !strings.Contains(output, "Emoji Filter") || !strings.Contains(output, "global") {
		t.Errorf("expected enabled+global row, got:\n%s", output)
	}
	// Private, enabled
	if !strings.Contains(output, "Tone Filter") || !strings.Contains(output, "private") {
		t.Errorf("expected enabled+private row, got:\n%s", output)
	}
	// Disabled rows must show "-" as scope (only enabled functions get a scope label)
	if !strings.Contains(output, "disabled") {
		t.Errorf("expected disabled rows in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 4 function(s).") {
		t.Errorf("expected summary line, got:\n%s", output)
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	server := newFunctionsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "json")
	defer listCmd.Flags().Set("output", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var fns []api.Function
	if err := json.Unmarshal(buf.Bytes(), &fns); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\nOutput:\n%s", err, buf.String())
	}
	if len(fns) != 4 {
		t.Errorf("expected 4 functions in JSON, got %d", len(fns))
	}
}

func TestListCommand_FilterEnabled(t *testing.T) {
	server := newFunctionsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	listCmd.Flags().Set("filter", "enabled")
	defer listCmd.Flags().Set("filter", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Showing 2 function(s).") {
		t.Errorf("expected 2 enabled functions, got:\n%s", output)
	}
	if strings.Contains(output, "Draft Action") || strings.Contains(output, "Private Pipe") {
		t.Errorf("disabled functions leaked into enabled filter:\n%s", output)
	}
}

func TestListCommand_FilterGlobal(t *testing.T) {
	server := newFunctionsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	listCmd.Flags().Set("filter", "global")
	defer listCmd.Flags().Set("filter", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Emoji Filter") {
		t.Errorf("expected Emoji Filter (enabled+global), got:\n%s", output)
	}
	// Private Pipe has is_global=true but is_active=false, so must NOT appear.
	if strings.Contains(output, "Private Pipe") {
		t.Errorf("inactive global function leaked into 'global' filter:\n%s", output)
	}
	if !strings.Contains(output, "Showing 1 function(s).") {
		t.Errorf("expected exactly 1 enabled+global function, got:\n%s", output)
	}
}

func TestListCommand_FilterPrivate(t *testing.T) {
	server := newFunctionsServer(t)
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
	if !strings.Contains(output, "Tone Filter") {
		t.Errorf("expected Tone Filter (enabled+private), got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 1 function(s).") {
		t.Errorf("expected exactly 1 enabled+private function, got:\n%s", output)
	}
}

func TestListCommand_FilterByType(t *testing.T) {
	server := newFunctionsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	resetListFlags(t)
	listCmd.Flags().Set("type", "action")
	defer listCmd.Flags().Set("type", "")

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Draft Action") {
		t.Errorf("expected Draft Action in type=action filter, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 1 function(s).") {
		t.Errorf("expected exactly 1 action, got:\n%s", output)
	}
}

func TestListCommand_InvalidFilter(t *testing.T) {
	server := newFunctionsServer(t)
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

func TestFunctionScopeAndStatus(t *testing.T) {
	cases := []struct {
		f          api.Function
		wantStatus string
		wantScope  string
	}{
		{api.Function{IsActive: true, IsGlobal: true}, "enabled", "global"},
		{api.Function{IsActive: true, IsGlobal: false}, "enabled", "private"},
		{api.Function{IsActive: false, IsGlobal: true}, "disabled", "-"},
		{api.Function{IsActive: false, IsGlobal: false}, "disabled", "-"},
	}
	for i, tc := range cases {
		if got := functionStatus(tc.f); got != tc.wantStatus {
			t.Errorf("case %d: functionStatus = %q, want %q", i, got, tc.wantStatus)
		}
		if got := functionScope(tc.f); got != tc.wantScope {
			t.Errorf("case %d: functionScope = %q, want %q", i, got, tc.wantScope)
		}
	}
}

// resetListFlags clears any flag state lingering from prior subtests so each
// case sees the same starting point. Cobra commands are package-global here,
// mirroring the pattern already used in other CLI test suites.
//
// "filter" and "output" are persistent root flags in production but absent in
// unit tests; register local stand-ins so GetString() calls in list.go work.
func resetListFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"filter", "output"} {
		if listCmd.Flags().Lookup(name) == nil {
			listCmd.Flags().String(name, "", "")
		}
	}
	for _, name := range []string{"filter", "type", "output"} {
		if f := listCmd.Flags().Lookup(name); f != nil {
			listCmd.Flags().Set(name, "")
		}
	}
}

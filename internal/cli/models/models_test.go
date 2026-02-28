package models

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

// setupTestConfig creates a temp config directory pointing the active instance at the given server URL.
func setupTestConfig(t *testing.T, serverURL string) (cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "owui-models-test-*")
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

// setupEmptyConfig creates a temp config directory with no instances.
func setupEmptyConfig(t *testing.T) (cleanup func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "owui-models-test-empty-*")
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

// newModelsServer returns an httptest.Server that handles model, group, and user API endpoints.
func newModelsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/models/list":
			json.NewEncoder(w).Encode(api.ModelAccessListResponse{
				Items: []api.ModelAccessResponse{
					{
						ID: "gpt-4o", Name: "GPT-4o", BaseModelID: "openai/gpt-4o",
						IsActive: true, AccessGrants: []api.AccessGrantModel{},
						Meta:      api.ModelMeta{Description: "GPT-4o model"},
						UpdatedAt: 1708900000,
					},
					{
						ID: "claude-sonnet", Name: "Claude Sonnet", BaseModelID: "anthropic/claude-3.5",
						IsActive: true,
						AccessGrants: []api.AccessGrantModel{
							{ID: "grant-1", PrincipalType: "group", PrincipalID: "g1", Permission: "read", CreatedAt: 1706000000},
							{ID: "grant-2", PrincipalType: "group", PrincipalID: "g2", Permission: "read", CreatedAt: 1707000000},
						},
						Meta: api.ModelMeta{
							Description: "Claude 3.5 Sonnet",
							Capabilities: api.ModelCapabilities{
								boolPtr(true),
								boolPtr(true),
								boolPtr(false),
							},
						},
						User:      &api.ModelUser{ID: "u1", Name: "admin", Email: "admin@example.com"},
						UpdatedAt: 1708265520, CreatedAt: 1705312200,
					},
					{
						ID: "llama-3.1", Name: "Llama 3.1", BaseModelID: "ollama/llama3.1",
						IsActive: false, AccessGrants: []api.AccessGrantModel{},
						UpdatedAt: 1707600000,
					},
					{
						ID: "custom-assistant", Name: "Custom Assistant", BaseModelID: "gpt-4o",
						IsActive: true,
						AccessGrants: []api.AccessGrantModel{
							{ID: "grant-3", PrincipalType: "group", PrincipalID: "g1", Permission: "read", CreatedAt: 1708000000},
						},
						UpdatedAt: 1709000000,
					},
				},
				Total: 4,
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/models/model":
			id := r.URL.Query().Get("id")
			switch id {
			case "claude-sonnet":
				json.NewEncoder(w).Encode(api.ModelAccessResponse{
					ID: "claude-sonnet", Name: "Claude Sonnet", BaseModelID: "anthropic/claude-3.5-sonnet",
					IsActive: true,
					AccessGrants: []api.AccessGrantModel{
						{ID: "grant-1", PrincipalType: "group", PrincipalID: "g1", Permission: "read", CreatedAt: 1706000000},
						{ID: "grant-2", PrincipalType: "group", PrincipalID: "g2", Permission: "read", CreatedAt: 1707000000},
						{ID: "grant-3", PrincipalType: "group", PrincipalID: "g3", Permission: "read", CreatedAt: 1707600000},
					},
					Meta: api.ModelMeta{
						Description:  "Claude 3.5 Sonnet by Anthropic",
						Capabilities: api.ModelCapabilities{boolPtr(true), boolPtr(true), boolPtr(false)},
					},
					User:      &api.ModelUser{ID: "u1", Name: "admin", Email: "admin@example.com"},
					UpdatedAt: 1708265520, CreatedAt: 1705312200,
				})
			case "gpt-4o":
				json.NewEncoder(w).Encode(api.ModelAccessResponse{
					ID: "gpt-4o", Name: "GPT-4o", BaseModelID: "openai/gpt-4o",
					IsActive: true, AccessGrants: []api.AccessGrantModel{},
					Meta: api.ModelMeta{Description: "GPT-4o model"},
				})
			default:
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"detail":"model not found"}`))
			}
		case r.Method == "POST" && r.URL.Path == "/api/v1/models/model/toggle":
			id := r.URL.Query().Get("id")
			json.NewEncoder(w).Encode(api.ModelResponse{
				ID: id, Name: "Model", IsActive: false,
			})
		case r.Method == "POST" && r.URL.Path == "/api/v1/models/model/access/update":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/groups/":
			json.NewEncoder(w).Encode([]api.Group{
				{ID: "g1", Name: "developers", Description: "Dev team"},
				{ID: "g2", Name: "backend-team", Description: "Backend team"},
				{ID: "g3", Name: "designers", Description: "Design team"},
				{ID: "g4", Name: "oauth-group", Description: "Group oauth-group created automatically via OAuth."},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"detail":"not found"}`))
		}
	}))
}

func boolPtr(v bool) *bool {
	return &v
}

// --- list command tests ---

func TestListCommand_PrettyOutput(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	for _, expected := range []string{"NAME", "ID", "BASE MODEL", "STATUS", "VISIBILITY", "GRANTS", "UPDATED"} {
		if !strings.Contains(output, expected) {
			t.Errorf("expected header %q in output, got:\n%s", expected, output)
		}
	}
	if !strings.Contains(output, "GPT-4o") {
		t.Errorf("expected 'GPT-4o' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "claude-sonnet") {
		t.Errorf("expected 'claude-sonnet' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "enabled") {
		t.Errorf("expected 'enabled' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "disabled") {
		t.Errorf("expected 'disabled' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "public") {
		t.Errorf("expected 'public' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "private") {
		t.Errorf("expected 'private' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 4 model(s).") {
		t.Errorf("expected summary line, got:\n%s", output)
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "json")
	defer listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var models []api.ModelAccessResponse
	if err := json.Unmarshal([]byte(output), &models); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nOutput:\n%s", err, output)
	}
	if len(models) != 4 {
		t.Errorf("expected 4 models in JSON, got %d", len(models))
	}
}

func TestListCommand_FilterEnabled(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "enabled")
	defer listCmd.Flags().Set("filter", "")

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Llama 3.1") {
		t.Errorf("expected disabled model 'Llama 3.1' to be filtered out, got:\n%s", output)
	}
	if !strings.Contains(output, "GPT-4o") {
		t.Errorf("expected enabled model 'GPT-4o', got:\n%s", output)
	}
	if !strings.Contains(output, `Showing 3 model(s) matching filter "enabled".`) {
		t.Errorf("expected filter summary line, got:\n%s", output)
	}
}

func TestListCommand_FilterDisabled(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "disabled")
	defer listCmd.Flags().Set("filter", "")

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Llama 3.1") {
		t.Errorf("expected 'Llama 3.1' in output, got:\n%s", output)
	}
	if strings.Contains(output, "GPT-4o") {
		t.Errorf("expected enabled models to be filtered out, got:\n%s", output)
	}
	if !strings.Contains(output, `Showing 1 model(s) matching filter "disabled".`) {
		t.Errorf("expected filter summary line, got:\n%s", output)
	}
}

func TestListCommand_FilterPrivate(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "private")
	defer listCmd.Flags().Set("filter", "")

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Claude Sonnet") {
		t.Errorf("expected private model 'Claude Sonnet', got:\n%s", output)
	}
	if !strings.Contains(output, "Custom Assistant") {
		t.Errorf("expected private model 'Custom Assistant', got:\n%s", output)
	}
	if strings.Contains(output, "GPT-4o") {
		t.Errorf("expected public model 'GPT-4o' to be filtered out, got:\n%s", output)
	}
	if !strings.Contains(output, `Showing 2 model(s) matching filter "private".`) {
		t.Errorf("expected filter summary line, got:\n%s", output)
	}
}

func TestListCommand_InvalidFilter(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "invalid")
	defer listCmd.Flags().Set("filter", "")

	err := listCmd.RunE(listCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid filter")
	}
	if !strings.Contains(err.Error(), "invalid filter") {
		t.Errorf("expected 'invalid filter' error, got: %v", err)
	}
}

func TestListCommand_EmptyModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.ModelAccessListResponse{Items: []api.ModelAccessResponse{}, Total: 0})
	}))
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "")

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No models found") {
		t.Errorf("expected 'No models found' message, got:\n%s", output)
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

func TestListCommand_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"server error"}`))
	}))
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "")

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestListCommand_SortedByName(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "")

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	claudeIdx := strings.Index(output, "Claude Sonnet")
	customIdx := strings.Index(output, "Custom Assistant")
	gptIdx := strings.Index(output, "GPT-4o")
	llamaIdx := strings.Index(output, "Llama 3.1")
	if claudeIdx > customIdx || customIdx > gptIdx || gptIdx > llamaIdx {
		t.Errorf("expected models sorted alphabetically, got:\n%s", output)
	}
}

// --- show command tests ---

func TestShowCommand_PrettyOutput(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	showCmd.SetOut(buf)

	if showCmd.Flags().Lookup("output") == nil {
		showCmd.Flags().String("output", "", "")
	}
	showCmd.Flags().Set("output", "")

	err := showCmd.RunE(showCmd, []string{"claude-sonnet"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Claude Sonnet") {
		t.Errorf("expected 'Claude Sonnet' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "claude-sonnet") {
		t.Errorf("expected 'claude-sonnet' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "General") {
		t.Errorf("expected 'General' section, got:\n%s", output)
	}
	if !strings.Contains(output, "Status") {
		t.Errorf("expected 'Status' section, got:\n%s", output)
	}
	if !strings.Contains(output, "Capabilities") {
		t.Errorf("expected 'Capabilities' section, got:\n%s", output)
	}
	if !strings.Contains(output, "Access Grants") {
		t.Errorf("expected 'Access Grants' section, got:\n%s", output)
	}
	if !strings.Contains(output, "developers") {
		t.Errorf("expected resolved group name 'developers', got:\n%s", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("expected owner 'admin', got:\n%s", output)
	}
}

func TestShowCommand_JSONOutput(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	showCmd.SetOut(buf)

	if showCmd.Flags().Lookup("output") == nil {
		showCmd.Flags().String("output", "", "")
	}
	showCmd.Flags().Set("output", "json")
	defer showCmd.Flags().Set("output", "")

	err := showCmd.RunE(showCmd, []string{"claude-sonnet"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nOutput:\n%s", err, output)
	}
	if _, ok := result["id"]; !ok {
		t.Error("expected 'id' key in JSON output")
	}
	if _, ok := result["access_grants"]; !ok {
		t.Error("expected 'access_grants' key in JSON output")
	}
}

func TestShowCommand_ModelNotFound(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	showCmd.SetOut(buf)

	err := showCmd.RunE(showCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestShowCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	showCmd.SetOut(buf)

	err := showCmd.RunE(showCmd, []string{"gpt-4o"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestShowCommand_PublicModelNoGrants(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	showCmd.SetOut(buf)

	if showCmd.Flags().Lookup("output") == nil {
		showCmd.Flags().String("output", "", "")
	}
	showCmd.Flags().Set("output", "")

	err := showCmd.RunE(showCmd, []string{"gpt-4o"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No access grants") {
		t.Errorf("expected 'No access grants' message for public model, got:\n%s", output)
	}
}

// --- set-status command tests ---

func TestSetStatusCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	setStatusCmd.SetOut(buf)

	err := setStatusCmd.RunE(setStatusCmd, []string{"enable", "gpt-4o"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestSetStatusCommand_InvalidAction(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	setStatusCmd.SetOut(buf)

	err := setStatusCmd.RunE(setStatusCmd, []string{"invalid"})
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Errorf("expected 'invalid action' error, got: %v", err)
	}
}

func TestSetStatusCommand_NoAction(t *testing.T) {
	buf := new(bytes.Buffer)
	setStatusCmd.SetOut(buf)

	err := setStatusCmd.RunE(setStatusCmd, []string{})
	if err == nil {
		t.Fatal("expected error for missing action")
	}
	if !strings.Contains(err.Error(), "action required") {
		t.Errorf("expected 'action required' error, got: %v", err)
	}
}

func TestSetStatusCommand_ModelNotFound(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	setStatusCmd.SetOut(buf)

	err := setStatusCmd.RunE(setStatusCmd, []string{"enable", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- set-visibility command tests ---

func TestSetVisibilityCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	setVisibilityCmd.SetOut(buf)

	err := setVisibilityCmd.RunE(setVisibilityCmd, []string{"public", "claude-sonnet"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestSetVisibilityCommand_InvalidVisibility(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	setVisibilityCmd.SetOut(buf)

	err := setVisibilityCmd.RunE(setVisibilityCmd, []string{"invalid"})
	if err == nil {
		t.Fatal("expected error for invalid visibility")
	}
	if !strings.Contains(err.Error(), "invalid visibility") {
		t.Errorf("expected 'invalid visibility' error, got: %v", err)
	}
}

func TestSetVisibilityCommand_NoVisibility(t *testing.T) {
	buf := new(bytes.Buffer)
	setVisibilityCmd.SetOut(buf)

	err := setVisibilityCmd.RunE(setVisibilityCmd, []string{})
	if err == nil {
		t.Fatal("expected error for missing visibility")
	}
	if !strings.Contains(err.Error(), "visibility required") {
		t.Errorf("expected 'visibility required' error, got: %v", err)
	}
}

func TestSetVisibilityCommand_ModelNotFound(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	setVisibilityCmd.SetOut(buf)

	err := setVisibilityCmd.RunE(setVisibilityCmd, []string{"public", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- helper function tests ---

func TestFindModelByID(t *testing.T) {
	models := []api.ModelAccessResponse{
		{ID: "gpt-4o", Name: "GPT-4o"},
		{ID: "claude-sonnet", Name: "Claude Sonnet"},
	}

	t.Run("found", func(t *testing.T) {
		m, err := findModelByID(models, "claude-sonnet")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Name != "Claude Sonnet" {
			t.Errorf("expected Name 'Claude Sonnet', got %q", m.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findModelByID(models, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestFindModelByNameOrID(t *testing.T) {
	models := []api.ModelAccessResponse{
		{ID: "gpt-4o", Name: "GPT-4o"},
		{ID: "claude-sonnet", Name: "Claude Sonnet"},
	}

	t.Run("found by id", func(t *testing.T) {
		m, err := findModelByNameOrID(models, "gpt-4o")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.ID != "gpt-4o" {
			t.Errorf("expected ID 'gpt-4o', got %q", m.ID)
		}
	})

	t.Run("found by name", func(t *testing.T) {
		m, err := findModelByNameOrID(models, "Claude Sonnet")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.ID != "claude-sonnet" {
			t.Errorf("expected ID 'claude-sonnet', got %q", m.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findModelByNameOrID(models, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestIsPublic(t *testing.T) {
	public := api.ModelAccessResponse{ID: "m1", AccessGrants: []api.AccessGrantModel{}}
	if !isPublic(public) {
		t.Error("expected model with no grants to be public")
	}

	private := api.ModelAccessResponse{ID: "m2", AccessGrants: []api.AccessGrantModel{{PrincipalType: "group", PrincipalID: "g1"}}}
	if isPublic(private) {
		t.Error("expected model with grants to not be public")
	}
}

func TestIsPrivate(t *testing.T) {
	public := api.ModelAccessResponse{ID: "m1", AccessGrants: []api.AccessGrantModel{}}
	if isPrivate(public) {
		t.Error("expected model with no grants to not be private")
	}

	private := api.ModelAccessResponse{ID: "m2", AccessGrants: []api.AccessGrantModel{{PrincipalType: "group", PrincipalID: "g1"}}}
	if !isPrivate(private) {
		t.Error("expected model with grants to be private")
	}
}

func TestModelStatus(t *testing.T) {
	enabled := api.ModelAccessResponse{IsActive: true}
	if modelStatus(enabled) != "enabled" {
		t.Errorf("expected 'enabled', got %q", modelStatus(enabled))
	}

	disabled := api.ModelAccessResponse{IsActive: false}
	if modelStatus(disabled) != "disabled" {
		t.Errorf("expected 'disabled', got %q", modelStatus(disabled))
	}
}

func TestModelVisibility(t *testing.T) {
	public := api.ModelAccessResponse{AccessGrants: []api.AccessGrantModel{}}
	if modelVisibility(public) != "public" {
		t.Errorf("expected 'public', got %q", modelVisibility(public))
	}

	private := api.ModelAccessResponse{AccessGrants: []api.AccessGrantModel{{PrincipalType: "group", PrincipalID: "g1"}}}
	if modelVisibility(private) != "private" {
		t.Errorf("expected 'private', got %q", modelVisibility(private))
	}
}

// --- add-to-group command tests ---

func TestAddToGroupCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	addToGroupCmd.SetOut(buf)

	err := addToGroupCmd.RunE(addToGroupCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

// --- remove-from-group command tests ---

func TestRemoveFromGroupCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	removeFromGroupCmd.SetOut(buf)

	err := removeFromGroupCmd.RunE(removeFromGroupCmd, []string{"claude-sonnet"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

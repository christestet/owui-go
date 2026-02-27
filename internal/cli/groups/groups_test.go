package groups

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

	tmpDir, err := os.MkdirTemp("", "owui-groups-test-*")
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

	tmpDir, err := os.MkdirTemp("", "owui-groups-test-empty-*")
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

func memberCountPtr(n int) *int {
	return &n
}

// newGroupsServer returns an httptest.Server that handles group and user API endpoints.
func newGroupsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/groups/":
			json.NewEncoder(w).Encode([]api.Group{
				{ID: "g1", Name: "developers", Description: "Dev team", MemberCount: memberCountPtr(3)},
				{ID: "g2", Name: "oauth-group", Description: "Group oauth-group created automatically via OAuth.", MemberCount: memberCountPtr(5)},
				{ID: "g3", Name: "designers", Description: "Design team", MemberCount: memberCountPtr(2)},
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/groups/id/g1":
			json.NewEncoder(w).Encode(api.Group{
				ID:          "g1",
				Name:        "developers",
				Description: "Dev team",
				MemberCount: memberCountPtr(3),
				CreatedAt:   1700000000,
				UpdatedAt:   1700100000,
			})
		case r.Method == "POST" && r.URL.Path == "/api/v1/groups/create":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.Group{ID: "g-new", Name: "new-group", Description: "New group"})
		case r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/v1/groups/id/") && strings.HasSuffix(r.URL.Path, "/delete"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`true`))
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/update"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(api.Group{ID: "g1", Name: "updated-developers", Description: "Updated team"})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/users") && !strings.HasSuffix(r.URL.Path, "/users/add") && !strings.HasSuffix(r.URL.Path, "/users/remove"):
			// GetGroupMembers
			json.NewEncoder(w).Encode([]api.User{
				{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "user"},
				{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
			})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/users/add"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"g1","name":"developers"}`))
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/users/remove"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"g1","name":"developers"}`))
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/export"):
			json.NewEncoder(w).Encode(api.Group{
				ID:      "g1",
				Name:    "developers",
				UserIDs: []string{"u1", "u2"},
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/users/":
			json.NewEncoder(w).Encode(api.UserListResponse{
				Users: []api.User{
					{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "user"},
					{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
					{ID: "u3", Name: "charlie", Email: "charlie@example.com", Role: "admin"},
					{ID: "u4", Name: "diana", Email: "diana@example.com", Role: "pending"},
				},
				Total: 4,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"detail":"not found"}`))
		}
	}))
}

// --- list command tests ---

func TestListCommand_PrettyOutput(t *testing.T) {
	server := newGroupsServer(t)
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
	if !strings.Contains(output, "NAME") || !strings.Contains(output, "DESCRIPTION") || !strings.Contains(output, "MEMBERS") || !strings.Contains(output, "TYPE") {
		t.Errorf("expected table headers, got:\n%s", output)
	}
	if !strings.Contains(output, "developers") {
		t.Errorf("expected 'developers' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "oauth-group") {
		t.Errorf("expected 'oauth-group' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "local") {
		t.Errorf("expected 'local' type in output, got:\n%s", output)
	}
	if !strings.Contains(output, "oauth") {
		t.Errorf("expected 'oauth' type in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 3 group(s).") {
		t.Errorf("expected summary line, got:\n%s", output)
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	server := newGroupsServer(t)
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
	var groups []api.Group
	if err := json.Unmarshal([]byte(output), &groups); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nOutput:\n%s", err, output)
	}
	if len(groups) != 3 {
		t.Errorf("expected 3 groups in JSON, got %d", len(groups))
	}
}

func TestListCommand_FilterLocal(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "local")
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
	if !strings.Contains(output, "developers") {
		t.Errorf("expected 'developers' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "designers") {
		t.Errorf("expected 'designers' in output, got:\n%s", output)
	}
	if strings.Contains(output, "oauth-group") {
		t.Errorf("expected 'oauth-group' to be filtered out, got:\n%s", output)
	}
	if !strings.Contains(output, `Showing 2 group(s) matching filter "local".`) {
		t.Errorf("expected filter summary line, got:\n%s", output)
	}
}

func TestListCommand_FilterOAuth(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "oauth")
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
	if !strings.Contains(output, "oauth-group") {
		t.Errorf("expected 'oauth-group' in output, got:\n%s", output)
	}
	if strings.Contains(output, "developers") {
		t.Errorf("expected 'developers' to be filtered out, got:\n%s", output)
	}
	if !strings.Contains(output, `Showing 1 group(s) matching filter "oauth".`) {
		t.Errorf("expected filter summary line, got:\n%s", output)
	}
}

func TestListCommand_InvalidFilter(t *testing.T) {
	server := newGroupsServer(t)
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

func TestListCommand_EmptyGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Group{})
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
	if !strings.Contains(output, "No groups found") {
		t.Errorf("expected 'No groups found' message, got:\n%s", output)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]api.Group{
			{ID: "g3", Name: "zebra-team", Description: "Z team"},
			{ID: "g1", Name: "alpha-team", Description: "A team"},
			{ID: "g2", Name: "beta-team", Description: "B team"},
		})
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
	alphaIdx := strings.Index(output, "alpha-team")
	betaIdx := strings.Index(output, "beta-team")
	zebraIdx := strings.Index(output, "zebra-team")
	if alphaIdx > betaIdx || betaIdx > zebraIdx {
		t.Errorf("expected groups sorted alphabetically, got:\n%s", output)
	}
}

// --- add command tests ---

func TestAddCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	addCmd.SetOut(buf)

	addCmd.Flags().Set("name", "test")
	addCmd.Flags().Set("description", "test desc")
	defer func() {
		addCmd.Flags().Set("name", "")
		addCmd.Flags().Set("description", "")
	}()

	err := addCmd.RunE(addCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestAddCommand_NonInteractive(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	addCmd.SetOut(buf)

	addCmd.Flags().Set("name", "new-team")
	addCmd.Flags().Set("description", "A new team")
	defer func() {
		addCmd.Flags().Set("name", "")
		addCmd.Flags().Set("description", "")
	}()

	err := addCmd.RunE(addCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Successfully created group") {
		t.Errorf("expected success message, got:\n%s", output)
	}
}

func TestAddCommand_InvalidPermissions(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	addCmd.SetOut(buf)

	addCmd.Flags().Set("name", "test")
	addCmd.Flags().Set("description", "test")
	addCmd.Flags().Set("permissions", "not-valid-json")
	defer func() {
		addCmd.Flags().Set("name", "")
		addCmd.Flags().Set("description", "")
		addCmd.Flags().Set("permissions", "")
	}()

	err := addCmd.RunE(addCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid permissions JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error, got: %v", err)
	}
}

// --- remove command tests ---

func TestRemoveCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	removeCmd.SetOut(buf)

	err := removeCmd.RunE(removeCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestRemoveCommand_GroupNotFound(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	removeCmd.SetOut(buf)

	err := removeCmd.RunE(removeCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- update command tests ---

func TestUpdateCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)

	err := updateCmd.RunE(updateCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestUpdateCommand_GroupNotFound(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)

	err := updateCmd.RunE(updateCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- members command tests ---

func TestMembersCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	membersCmd.SetOut(buf)

	err := membersCmd.RunE(membersCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestMembersCommand_GroupNotFound(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	membersCmd.SetOut(buf)

	err := membersCmd.RunE(membersCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestMembersCommand_PrettyOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	membersCmd.SetOut(buf)

	if membersCmd.Flags().Lookup("output") == nil {
		membersCmd.Flags().String("output", "", "")
	}
	membersCmd.Flags().Set("output", "")

	err := membersCmd.RunE(membersCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Group: developers") {
		t.Errorf("expected 'Group: developers' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Description: Dev team") {
		t.Errorf("expected description in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Type: local") {
		t.Errorf("expected 'Type: local' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "alice") {
		t.Errorf("expected 'alice' in members output, got:\n%s", output)
	}
	if !strings.Contains(output, "bob") {
		t.Errorf("expected 'bob' in members output, got:\n%s", output)
	}
	if !strings.Contains(output, "NAME") || !strings.Contains(output, "EMAIL") || !strings.Contains(output, "ROLE") {
		t.Errorf("expected member table headers, got:\n%s", output)
	}
}

func TestMembersCommand_JSONOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	membersCmd.SetOut(buf)

	if membersCmd.Flags().Lookup("output") == nil {
		membersCmd.Flags().String("output", "", "")
	}
	membersCmd.Flags().Set("output", "json")
	defer membersCmd.Flags().Set("output", "")

	err := membersCmd.RunE(membersCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nOutput:\n%s", err, output)
	}
	if _, ok := result["group"]; !ok {
		t.Error("expected 'group' key in JSON output")
	}
	if _, ok := result["members"]; !ok {
		t.Error("expected 'members' key in JSON output")
	}
}

// --- add-users command tests ---

func TestAddUsersCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	addUsersCmd.SetOut(buf)

	err := addUsersCmd.RunE(addUsersCmd, []string{"alice"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestAddUsersCommand_NonUserRole(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	addUsersCmd.SetOut(buf)

	if addUsersCmd.Flags().Lookup("group") == nil {
		addUsersCmd.Flags().String("group", "", "")
	}
	addUsersCmd.Flags().Set("group", "developers")
	defer addUsersCmd.Flags().Set("group", "")

	// charlie is an admin, should be rejected
	err := addUsersCmd.RunE(addUsersCmd, []string{"charlie"})
	if err == nil {
		t.Fatal("expected error when adding admin user to group")
	}
	if !strings.Contains(err.Error(), "role") {
		t.Errorf("expected role-related error, got: %v", err)
	}
}

// --- remove-users command tests ---

func TestRemoveUsersCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	removeUsersCmd.SetOut(buf)

	err := removeUsersCmd.RunE(removeUsersCmd, []string{"alice"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

// --- helper function tests ---

func TestIsOAuthGroup(t *testing.T) {
	tests := []struct {
		name     string
		group    api.Group
		expected bool
	}{
		{
			name:     "oauth group",
			group:    api.Group{Name: "oauth-team", Description: "Group oauth-team created automatically via OAuth."},
			expected: true,
		},
		{
			name:     "local group",
			group:    api.Group{Name: "developers", Description: "Dev team"},
			expected: false,
		},
		{
			name:     "empty description",
			group:    api.Group{Name: "test", Description: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOAuthGroup(tt.group); got != tt.expected {
				t.Errorf("isOAuthGroup(%q) = %v, want %v", tt.group.Description, got, tt.expected)
			}
		})
	}
}

func TestGroupType(t *testing.T) {
	local := api.Group{Name: "devs", Description: "Dev team"}
	if groupType(local) != "local" {
		t.Errorf("expected 'local', got %q", groupType(local))
	}

	oauth := api.Group{Name: "eng", Description: "Group eng created automatically via OAuth."}
	if groupType(oauth) != "oauth" {
		t.Errorf("expected 'oauth', got %q", groupType(oauth))
	}
}

func TestFilterLocalGroups(t *testing.T) {
	groups := []api.Group{
		{ID: "g1", Name: "developers", Description: "Dev team"},
		{ID: "g2", Name: "oauth-team", Description: "Group oauth-team created automatically via OAuth."},
		{ID: "g3", Name: "designers", Description: "Design team"},
	}

	local := filterLocalGroups(groups)
	if len(local) != 2 {
		t.Fatalf("expected 2 local groups, got %d", len(local))
	}
	for _, g := range local {
		if g.Name == "oauth-team" {
			t.Error("expected oauth-team to be filtered out")
		}
	}
}

func TestFilterOAuthGroups(t *testing.T) {
	groups := []api.Group{
		{ID: "g1", Name: "developers", Description: "Dev team"},
		{ID: "g2", Name: "oauth-team", Description: "Group oauth-team created automatically via OAuth."},
		{ID: "g3", Name: "designers", Description: "Design team"},
	}

	oauth := filterOAuthGroups(groups)
	if len(oauth) != 1 {
		t.Fatalf("expected 1 oauth group, got %d", len(oauth))
	}
	if oauth[0].Name != "oauth-team" {
		t.Errorf("expected 'oauth-team', got %q", oauth[0].Name)
	}
}

func TestFindGroupByName(t *testing.T) {
	groups := []api.Group{
		{ID: "g1", Name: "developers", Description: "Dev team"},
		{ID: "g2", Name: "designers", Description: "Design team"},
	}

	t.Run("found", func(t *testing.T) {
		g, err := findGroupByName(groups, "designers")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if g.ID != "g2" {
			t.Errorf("expected ID 'g2', got %q", g.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findGroupByName(groups, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestFindUserByName(t *testing.T) {
	users := []api.User{
		{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "admin"},
		{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
	}

	t.Run("found", func(t *testing.T) {
		u, err := findUserByName(users, "bob")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.ID != "u2" {
			t.Errorf("expected ID 'u2', got %q", u.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findUserByName(users, "nobody")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestFilterUsersByRole(t *testing.T) {
	users := []api.User{
		{ID: "u1", Name: "alice", Role: "admin"},
		{ID: "u2", Name: "bob", Role: "user"},
		{ID: "u3", Name: "charlie", Role: "user"},
		{ID: "u4", Name: "diana", Role: "pending"},
	}

	filtered := filterUsersByRole(users, "user")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 users with role 'user', got %d", len(filtered))
	}
	for _, u := range filtered {
		if u.Role != "user" {
			t.Errorf("expected role 'user', got %q for %q", u.Role, u.Name)
		}
	}
}

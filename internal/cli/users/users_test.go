//go:build interactive
// +build interactive

package users

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

	tmpDir, err := os.MkdirTemp("", "owui-users-test-*")
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

	tmpDir, err := os.MkdirTemp("", "owui-users-test-empty-*")
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

// newUsersServer returns an httptest.Server that handles user and group API endpoints.
func newUsersServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/users/":
			allUsers := []api.User{
				{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "admin"},
				{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
				{ID: "u3", Name: "charlie", Email: "charlie@example.com", Role: "pending"},
			}
			query := r.URL.Query().Get("query")
			if query != "" {
				var filtered []api.User
				for _, u := range allUsers {
					if strings.Contains(u.Name, query) || strings.Contains(u.Email, query) {
						filtered = append(filtered, u)
					}
				}
				allUsers = filtered
			}
			json.NewEncoder(w).Encode(api.UserListResponse{
				Users: allUsers,
				Total: len(allUsers),
			})
		case r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/v1/users/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`true`))
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/update"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"u1","role":"admin"}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/auths/add":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token":"abc"}`))
		case r.Method == "GET" && r.URL.Path == "/api/v1/groups/":
			json.NewEncoder(w).Encode([]api.Group{
				{ID: "g1", Name: "developers", Description: "Dev team"},
				{ID: "g2", Name: "oauth-group", Description: "Group oauth-group created automatically via OAuth."},
				{ID: "g3", Name: "designers", Description: "Design team"},
			})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/users/add"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"g1","name":"developers"}`))
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/users/remove"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"g1","name":"developers"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"detail":"not found"}`))
		}
	}))
}

// --- list command tests ---

func TestListCommand_PrettyOutput(t *testing.T) {
	server := newUsersServer(t)
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
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "EMAIL") || !strings.Contains(output, "ROLE") {
		t.Errorf("expected table headers, got:\n%s", output)
	}
	if !strings.Contains(output, "alice") {
		t.Errorf("expected 'alice' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "bob") {
		t.Errorf("expected 'bob' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "charlie") {
		t.Errorf("expected 'charlie' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("expected 'admin' role in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 3 user(s).") {
		t.Errorf("expected summary line, got:\n%s", output)
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	// Register output flag if not present
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
	var users []api.User
	if err := json.Unmarshal([]byte(output), &users); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nOutput:\n%s", err, output)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users in JSON, got %d", len(users))
	}
}

func TestListCommand_EmptyUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.UserListResponse{Users: []api.User{}, Total: 0})
	}))
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	// Ensure pretty output
	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No users found") {
		t.Errorf("expected 'No users found' message, got:\n%s", output)
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

	// Ensure pretty output
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
		json.NewEncoder(w).Encode(api.UserListResponse{
			Users: []api.User{
				{ID: "u3", Name: "charlie", Email: "c@x.com", Role: "user"},
				{ID: "u1", Name: "alice", Email: "a@x.com", Role: "admin"},
				{ID: "u2", Name: "bob", Email: "b@x.com", Role: "user"},
			},
			Total: 3,
		})
	}))
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("output") == nil {
		listCmd.Flags().String("output", "", "")
	}
	listCmd.Flags().Set("output", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	aliceIdx := strings.Index(output, "alice")
	bobIdx := strings.Index(output, "bob")
	charlieIdx := strings.Index(output, "charlie")
	if aliceIdx > bobIdx || bobIdx > charlieIdx {
		t.Errorf("expected users sorted alphabetically, got:\n%s", output)
	}
}

func TestListCommand_FilterParam(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	listCmd.SetOut(buf)

	if listCmd.Flags().Lookup("filter") == nil {
		listCmd.Flags().String("filter", "", "")
	}
	listCmd.Flags().Set("filter", "alice")
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
	if !strings.Contains(output, "alice") {
		t.Errorf("expected 'alice' in filtered output, got:\n%s", output)
	}
	if strings.Contains(output, "bob") {
		t.Errorf("expected 'bob' to be filtered out, got:\n%s", output)
	}
	if !strings.Contains(output, `Showing 1 user(s) matching "alice".`) {
		t.Errorf("expected filter summary line, got:\n%s", output)
	}
}

func TestListCommand_RoleFilter(t *testing.T) {
	server := newUsersServer(t)
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

	listCmd.Flags().Set("role", "user")
	defer listCmd.Flags().Set("role", "")

	err := listCmd.RunE(listCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "bob") {
		t.Errorf("expected 'bob' (role user) in output, got:\n%s", output)
	}
	if strings.Contains(output, "alice") {
		t.Errorf("expected 'alice' (role admin) to be filtered out, got:\n%s", output)
	}
	if strings.Contains(output, "charlie") {
		t.Errorf("expected 'charlie' (role pending) to be filtered out, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 1 user(s).") {
		t.Errorf("expected summary line, got:\n%s", output)
	}
}

// --- remove command tests ---

func TestRemoveCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	removeCmd.SetOut(buf)

	err := removeCmd.RunE(removeCmd, []string{"alice"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestRemoveCommand_UserNotFound(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	removeCmd.SetOut(buf)

	err := removeCmd.RunE(removeCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- update-role command tests ---

func TestUpdateRoleCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	updateRoleCmd.SetOut(buf)

	err := updateRoleCmd.RunE(updateRoleCmd, []string{"alice"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestUpdateRoleCommand_UserNotFound(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	updateRoleCmd.SetOut(buf)

	err := updateRoleCmd.RunE(updateRoleCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- create command tests ---

func TestCreateCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	createCmd.SetOut(buf)

	// Set all flags to skip interactive mode
	createCmd.Flags().Set("name", "testuser")
	createCmd.Flags().Set("email", "test@example.com")
	createCmd.Flags().Set("password", "pass123")
	createCmd.Flags().Set("role", "user")
	defer func() {
		createCmd.Flags().Set("name", "")
		createCmd.Flags().Set("email", "")
		createCmd.Flags().Set("password", "")
		createCmd.Flags().Set("role", "")
	}()

	err := createCmd.RunE(createCmd, []string{})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestCreateCommand_NonInteractive(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	createCmd.SetOut(buf)

	createCmd.Flags().Set("name", "newuser")
	createCmd.Flags().Set("email", "new@example.com")
	createCmd.Flags().Set("password", "pass123")
	createCmd.Flags().Set("role", "user")
	defer func() {
		createCmd.Flags().Set("name", "")
		createCmd.Flags().Set("email", "")
		createCmd.Flags().Set("password", "")
		createCmd.Flags().Set("role", "")
	}()

	err := createCmd.RunE(createCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Successfully created user newuser with role user") {
		t.Errorf("expected success message, got:\n%s", output)
	}
}

//go:build interactive
// +build interactive

func TestCreateCommand_MissingName(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	createCmd.SetOut(buf)

	// Set flags with missing name — but all others set to avoid interactive
	// Since name is empty, it will try to run wizard which will fail in test
	createCmd.Flags().Set("name", "")
	createCmd.Flags().Set("email", "test@example.com")
	createCmd.Flags().Set("password", "pass123")
	createCmd.Flags().Set("role", "user")
	defer func() {
		createCmd.Flags().Set("name", "")
		createCmd.Flags().Set("email", "")
		createCmd.Flags().Set("password", "")
		createCmd.Flags().Set("role", "")
	}()

	// The wizard will fail in a non-interactive test environment
	err := createCmd.RunE(createCmd, []string{})
	if err == nil {
		t.Fatal("expected error when interactive wizard can't run in test")
	}
}

// --- group command tests ---

func TestAddToGroupCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	addToGroupCmd.SetOut(buf)

	err := addToGroupCmd.RunE(addToGroupCmd, []string{"bob"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestRemoveFromGroupCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	removeFromGroupCmd.SetOut(buf)

	err := removeFromGroupCmd.RunE(removeFromGroupCmd, []string{"bob"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

// --- helper function tests ---

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

// --- add-to-group with non-user role test ---

func TestAddToGroupCommand_NonUserRole(t *testing.T) {
	server := newUsersServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	addToGroupCmd.SetOut(buf)

	// Register group flag if not present
	if addToGroupCmd.Flags().Lookup("group") == nil {
		addToGroupCmd.Flags().String("group", "", "")
	}
	addToGroupCmd.Flags().Set("group", "developers")
	defer addToGroupCmd.Flags().Set("group", "")

	// alice is an admin, should be rejected
	err := addToGroupCmd.RunE(addToGroupCmd, []string{"alice"})
	if err == nil {
		t.Fatal("expected error when adding admin user to group")
	}
	if !strings.Contains(err.Error(), "role") {
		t.Errorf("expected role-related error, got: %v", err)
	}
}

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
	"github.com/christestet/owui-go/internal/cli/shared"
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
		case r.Method == "GET" && r.URL.Path == "/api/v1/models/list":
			items := []api.ModelAccessResponse{
				{
					ID:       "m-alpha",
					Name:     "alpha",
					IsActive: true,
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
					},
				},
				{
					ID:       "m-beta",
					Name:     "beta",
					IsActive: true,
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g1", Permission: "write"},
					},
				},
				{
					ID:           "m-gamma",
					Name:         "gamma",
					IsActive:     true,
					AccessGrants: []api.AccessGrantModel{},
				},
				{
					ID:       "m-delta",
					Name:     "delta",
					IsActive: false,
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g2", Permission: "read"},
					},
				},
			}
			json.NewEncoder(w).Encode(api.ModelAccessListResponse{Items: items, Total: len(items)})
		case r.Method == "GET" && r.URL.Path == "/api/v1/tools/list":
			json.NewEncoder(w).Encode([]api.Tool{
				{
					ID:   "t-alpha",
					Name: "alpha-tool",
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
					},
				},
				{
					ID:   "t-beta",
					Name: "beta-tool",
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g1", Permission: "write"},
					},
				},
				{
					ID:           "t-gamma",
					Name:         "gamma-tool",
					AccessGrants: []api.AccessGrantModel{},
				},
				{
					ID:   "t-delta",
					Name: "delta-tool",
					AccessGrants: []api.AccessGrantModel{
						{PrincipalType: "group", PrincipalID: "g2", Permission: "read"},
					},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/groups/id/g1":
			json.NewEncoder(w).Encode(api.Group{
				ID:          "g1",
				Name:        "developers",
				Description: "Dev team",
				Permissions: json.RawMessage(`{"workspace":{"models":true,"knowledge":true,"prompts":false}}`),
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

func TestRemoveCommand_DuplicateGroupNameCannotDeleteWrongTarget(t *testing.T) {
	var deletedID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/":
			_ = json.NewEncoder(w).Encode([]api.Group{
				{ID: "g1", Name: "team", Description: "First"},
				{ID: "g2", Name: "team", Description: "Second"},
			})
		case r.Method == http.MethodDelete:
			deletedID = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/groups/id/"), "/delete")
			_, _ = w.Write([]byte(`true`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	removeCmd.SetOut(new(bytes.Buffer))
	if err := removeCmd.RunE(removeCmd, []string{"team"}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous remove error = %v, want ambiguity error", err)
	}
	if deletedID != "" {
		t.Fatalf("ambiguous group name deleted group %q", deletedID)
	}

	removeCmd.SetIn(strings.NewReader("y\n"))
	if err := removeCmd.RunE(removeCmd, []string{"g2"}); err != nil {
		t.Fatalf("ID-based remove error = %v", err)
	}
	if deletedID != "g2" {
		t.Fatalf("deleted group = %q, want g2", deletedID)
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

// --- show-permissions command tests ---

func TestShowPermissionsCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	showPermissionsCmd.SetOut(buf)

	err := showPermissionsCmd.RunE(showPermissionsCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestShowPermissionsCommand_GroupNotFound(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	showPermissionsCmd.SetOut(buf)

	err := showPermissionsCmd.RunE(showPermissionsCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestShowPermissionsCommand_PrettyOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	if showPermissionsCmd.Flags().Lookup("output") == nil {
		showPermissionsCmd.Flags().String("output", "", "")
	}
	showPermissionsCmd.Flags().Set("output", "")

	buf := new(bytes.Buffer)
	showPermissionsCmd.SetOut(buf)

	err := showPermissionsCmd.RunE(showPermissionsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Group: developers (g1)") {
		t.Errorf("expected group header, got:\n%s", output)
	}
	if !strings.Contains(output, "Permissions:") {
		t.Errorf("expected permissions header, got:\n%s", output)
	}
	if !strings.Contains(output, "PERMISSION") || !strings.Contains(output, "VALUE") {
		t.Errorf("expected permission table headers, got:\n%s", output)
	}
	if !strings.Contains(output, "workspace.models") || !strings.Contains(output, "true") {
		t.Errorf("expected models permission, got:\n%s", output)
	}
	if !strings.Contains(output, "workspace.prompts") || !strings.Contains(output, "false") {
		t.Errorf("expected prompts permission, got:\n%s", output)
	}
}

func TestShowPermissionsCommand_JSONOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	if showPermissionsCmd.Flags().Lookup("output") == nil {
		showPermissionsCmd.Flags().String("output", "", "")
	}
	showPermissionsCmd.Flags().Set("output", "json")
	defer showPermissionsCmd.Flags().Set("output", "")

	buf := new(bytes.Buffer)
	showPermissionsCmd.SetOut(buf)

	err := showPermissionsCmd.RunE(showPermissionsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Group struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"group"`
		Permissions map[string]map[string]bool `json:"permissions"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\nOutput:\n%s", err, buf.String())
	}
	if result.Group.ID != "g1" || result.Group.Name != "developers" {
		t.Errorf("unexpected group ref: %+v", result.Group)
	}
	if !result.Permissions["workspace"]["models"] {
		t.Errorf("expected workspace.models permission, got: %+v", result.Permissions)
	}
	if result.Permissions["workspace"]["prompts"] {
		t.Errorf("expected workspace.prompts to be false, got: %+v", result.Permissions)
	}
}

// --- add-users command tests ---

func TestFilterAddableUsers(t *testing.T) {
	tests := []struct {
		name            string
		users           []api.User
		existingUserIDs []string
		wantIDs         []string
	}{
		{
			name: "filters existing members and non-user roles",
			users: []api.User{
				{ID: "u1", Name: "alice", Role: "user"},
				{ID: "u2", Name: "bob", Role: "user"},
				{ID: "u3", Name: "charlie", Role: "admin"},
			},
			existingUserIDs: []string{"u1"},
			wantIDs:         []string{"u2"},
		},
		{
			name: "keeps all new users",
			users: []api.User{
				{ID: "u1", Name: "alice", Role: "user"},
				{ID: "u2", Name: "bob", Role: "user"},
			},
			wantIDs: []string{"u1", "u2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAddableUsers(tt.users, tt.existingUserIDs)
			gotIDs := make([]string, 0, len(got))
			for _, u := range got {
				gotIDs = append(gotIDs, u.ID)
			}
			if strings.Join(gotIDs, ",") != strings.Join(tt.wantIDs, ",") {
				t.Fatalf("filterAddableUsers() IDs = %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestUserIDOptions_UseHumanReadableLabelsAndUniqueIDs(t *testing.T) {
	users := []api.User{
		{ID: "u1", Name: "Alex", Email: "alex.one@example.com", Role: "user"},
		{ID: "u2", Name: "Alex", Email: "alex.two@example.com", Role: "user"},
	}

	options := userIDOptions(users)
	if len(options) != 2 {
		t.Fatalf("userIDOptions() returned %d options, want 2", len(options))
	}

	if options[0].Key != "Alex (alex.one@example.com)" || options[1].Key != "Alex (alex.two@example.com)" {
		t.Errorf("userIDOptions() labels = %q, %q; want human-readable name and email", options[0].Key, options[1].Key)
	}
	if options[0].Value != "u1" || options[1].Value != "u2" {
		t.Errorf("userIDOptions() values = %q, %q; want unique user IDs", options[0].Value, options[1].Value)
	}
}

func TestAddUsersCommand_FiltersExistingMembersFromBatch(t *testing.T) {
	var receivedUserIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/":
			if err := json.NewEncoder(w).Encode(api.UserListResponse{
				Users: []api.User{
					{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "user"},
					{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
				},
				Total: 2,
			}); err != nil {
				t.Errorf("encode users response: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/":
			if err := json.NewEncoder(w).Encode([]api.Group{{ID: "g1", Name: "developers"}}); err != nil {
				t.Errorf("encode groups response: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/id/g1/export":
			if err := json.NewEncoder(w).Encode(api.Group{ID: "g1", Name: "developers", UserIDs: []string{"u1"}}); err != nil {
				t.Errorf("encode group export response: %v", err)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/groups/id/g1/users/add":
			var form api.UserIdsForm
			if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
				t.Errorf("decode add-users request: %v", err)
			}
			receivedUserIDs = form.UserIDs
			if err := json.NewEncoder(w).Encode(api.Group{ID: "g1", Name: "developers"}); err != nil {
				t.Errorf("encode add-users response: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()

	buf := new(bytes.Buffer)
	addUsersCmd.SetIn(strings.NewReader("y\n"))
	addUsersCmd.SetOut(buf)
	defer addUsersCmd.SetIn(nil)
	defer addUsersCmd.SetOut(nil)

	if err := addUsersCmd.Flags().Set("group", "developers"); err != nil {
		t.Fatalf("set group flag: %v", err)
	}
	defer func() {
		if err := addUsersCmd.Flags().Set("group", ""); err != nil {
			t.Errorf("reset group flag: %v", err)
		}
	}()

	if err := addUsersCmd.RunE(addUsersCmd, []string{"alice", "bob", "bob"}); err != nil {
		t.Fatalf("addUsersCmd.RunE() error = %v", err)
	}

	if len(receivedUserIDs) != 1 || receivedUserIDs[0] != "u2" {
		t.Fatalf("add-users request user IDs = %v, want [u2]", receivedUserIDs)
	}
	if !strings.Contains(buf.String(), "Successfully added bob to group 'developers'") {
		t.Errorf("output = %q, want success message for newly added user only", buf.String())
	}
}

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

// --- show-models command tests ---

// resetShowModelsFlags sets the show-models flags back to defaults between tests.
func resetShowModelsFlags() {
	showModelsCmd.Flags().Set("permission", "all")
	showModelsCmd.Flags().Set("include-public", "false")
	if showModelsCmd.Flags().Lookup("output") != nil {
		showModelsCmd.Flags().Set("output", "")
	}
}

func TestShowModelsCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	showModelsCmd.SetOut(buf)

	err := showModelsCmd.RunE(showModelsCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestShowModelsCommand_GroupNotFound(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowModelsFlags()

	buf := new(bytes.Buffer)
	showModelsCmd.SetOut(buf)

	err := showModelsCmd.RunE(showModelsCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestShowModelsCommand_InvalidPermission(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowModelsFlags()

	buf := new(bytes.Buffer)
	showModelsCmd.SetOut(buf)
	showModelsCmd.Flags().Set("permission", "bogus")

	err := showModelsCmd.RunE(showModelsCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for invalid permission")
	}
	if !strings.Contains(err.Error(), "invalid permission") {
		t.Errorf("expected 'invalid permission' error, got: %v", err)
	}
}

func TestShowModelsCommand_PrettyOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowModelsFlags()

	if showModelsCmd.Flags().Lookup("output") == nil {
		showModelsCmd.Flags().String("output", "", "")
	}
	showModelsCmd.Flags().Set("output", "")
	showModelsCmd.Flags().Set("permission", "all")

	buf := new(bytes.Buffer)
	showModelsCmd.SetOut(buf)

	err := showModelsCmd.RunE(showModelsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Group: developers (g1)") {
		t.Errorf("expected group header, got:\n%s", output)
	}
	if !strings.Contains(output, "── Read ──") {
		t.Errorf("expected Read section header, got:\n%s", output)
	}
	if !strings.Contains(output, "── Write ──") {
		t.Errorf("expected Write section header, got:\n%s", output)
	}
	if !strings.Contains(output, "alpha") {
		t.Errorf("expected 'alpha' (read-granted) in output, got:\n%s", output)
	}
	if !strings.Contains(output, "beta") {
		t.Errorf("expected 'beta' (write-granted) in output, got:\n%s", output)
	}
	if strings.Contains(output, "delta") {
		t.Errorf("expected 'delta' (granted to g2) NOT to appear, got:\n%s", output)
	}
	if strings.Contains(output, "Public") {
		t.Errorf("expected no Public section without --include-public, got:\n%s", output)
	}
}

func TestShowModelsCommand_PermissionFilter(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowModelsFlags()

	if showModelsCmd.Flags().Lookup("output") == nil {
		showModelsCmd.Flags().String("output", "", "")
	}
	showModelsCmd.Flags().Set("output", "")
	showModelsCmd.Flags().Set("permission", "read")

	buf := new(bytes.Buffer)
	showModelsCmd.SetOut(buf)

	err := showModelsCmd.RunE(showModelsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "── Read ──") {
		t.Errorf("expected Read section, got:\n%s", output)
	}
	if strings.Contains(output, "── Write ──") {
		t.Errorf("expected no Write section with --permission read, got:\n%s", output)
	}
	if !strings.Contains(output, "alpha") {
		t.Errorf("expected 'alpha' in output, got:\n%s", output)
	}
	if strings.Contains(output, "beta") {
		t.Errorf("expected 'beta' (write) NOT in output, got:\n%s", output)
	}
}

func TestShowModelsCommand_IncludePublic(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowModelsFlags()

	if showModelsCmd.Flags().Lookup("output") == nil {
		showModelsCmd.Flags().String("output", "", "")
	}
	showModelsCmd.Flags().Set("output", "")
	showModelsCmd.Flags().Set("permission", "all")
	showModelsCmd.Flags().Set("include-public", "true")

	buf := new(bytes.Buffer)
	showModelsCmd.SetOut(buf)

	err := showModelsCmd.RunE(showModelsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Public") {
		t.Errorf("expected Public section, got:\n%s", output)
	}
	if !strings.Contains(output, "gamma") {
		t.Errorf("expected 'gamma' (public) in output, got:\n%s", output)
	}
}

func TestShowModelsCommand_JSONOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowModelsFlags()

	if showModelsCmd.Flags().Lookup("output") == nil {
		showModelsCmd.Flags().String("output", "", "")
	}
	showModelsCmd.Flags().Set("output", "json")
	showModelsCmd.Flags().Set("permission", "all")
	showModelsCmd.Flags().Set("include-public", "true")

	buf := new(bytes.Buffer)
	showModelsCmd.SetOut(buf)

	err := showModelsCmd.RunE(showModelsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Group struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"group"`
		Read   []api.ModelAccessResponse `json:"read"`
		Write  []api.ModelAccessResponse `json:"write"`
		Public []api.ModelAccessResponse `json:"public"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\nOutput:\n%s", err, buf.String())
	}
	if result.Group.ID != "g1" || result.Group.Name != "developers" {
		t.Errorf("unexpected group ref: %+v", result.Group)
	}
	if len(result.Read) != 1 || result.Read[0].ID != "m-alpha" {
		t.Errorf("expected one read model 'm-alpha', got: %+v", result.Read)
	}
	if len(result.Write) != 1 || result.Write[0].ID != "m-beta" {
		t.Errorf("expected one write model 'm-beta', got: %+v", result.Write)
	}
	if len(result.Public) != 1 || result.Public[0].ID != "m-gamma" {
		t.Errorf("expected one public model 'm-gamma', got: %+v", result.Public)
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
			if got := shared.IsOAuthGroup(tt.group); got != tt.expected {
				t.Errorf("shared.IsOAuthGroup(%q) = %v, want %v", tt.group.Description, got, tt.expected)
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

	local := shared.FilterLocalGroups(groups)
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
		g, err := shared.FindGroupByName(groups, "designers")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if g.ID != "g2" {
			t.Errorf("expected ID 'g2', got %q", g.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := shared.FindGroupByName(groups, "nonexistent")
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
		u, err := shared.FindUserByName(users, "bob")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.ID != "u2" {
			t.Errorf("expected ID 'u2', got %q", u.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := shared.FindUserByName(users, "nobody")
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

	filtered := shared.FilterUsersByRole(users, "user")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 users with role 'user', got %d", len(filtered))
	}
	for _, u := range filtered {
		if u.Role != "user" {
			t.Errorf("expected role 'user', got %q for %q", u.Role, u.Name)
		}
	}
}

// --- show-tools command tests ---

func resetShowToolsFlags() {
	showToolsCmd.Flags().Set("permission", "all")
	showToolsCmd.Flags().Set("include-public", "false")
	if showToolsCmd.Flags().Lookup("output") != nil {
		showToolsCmd.Flags().Set("output", "")
	}
}

func TestShowToolsCommand_NoActiveInstance(t *testing.T) {
	cleanup := setupEmptyConfig(t)
	defer cleanup()

	buf := new(bytes.Buffer)
	showToolsCmd.SetOut(buf)

	err := showToolsCmd.RunE(showToolsCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for no active instance")
	}
	if !strings.Contains(err.Error(), "no active instance") {
		t.Errorf("expected 'no active instance' error, got: %v", err)
	}
}

func TestShowToolsCommand_GroupNotFound(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowToolsFlags()

	buf := new(bytes.Buffer)
	showToolsCmd.SetOut(buf)

	err := showToolsCmd.RunE(showToolsCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestShowToolsCommand_InvalidPermission(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowToolsFlags()

	buf := new(bytes.Buffer)
	showToolsCmd.SetOut(buf)
	showToolsCmd.Flags().Set("permission", "bogus")

	err := showToolsCmd.RunE(showToolsCmd, []string{"developers"})
	if err == nil {
		t.Fatal("expected error for invalid permission")
	}
	if !strings.Contains(err.Error(), "invalid permission") {
		t.Errorf("expected 'invalid permission' error, got: %v", err)
	}
}

func TestShowToolsCommand_PrettyOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowToolsFlags()

	if showToolsCmd.Flags().Lookup("output") == nil {
		showToolsCmd.Flags().String("output", "", "")
	}
	showToolsCmd.Flags().Set("output", "")
	showToolsCmd.Flags().Set("permission", "all")

	buf := new(bytes.Buffer)
	showToolsCmd.SetOut(buf)

	err := showToolsCmd.RunE(showToolsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Group: developers (g1)") {
		t.Errorf("expected group header, got:\n%s", output)
	}
	if !strings.Contains(output, "── Read ──") {
		t.Errorf("expected Read section header, got:\n%s", output)
	}
	if !strings.Contains(output, "── Write ──") {
		t.Errorf("expected Write section header, got:\n%s", output)
	}
	if !strings.Contains(output, "alpha-tool") {
		t.Errorf("expected 'alpha-tool' (read-granted) in output, got:\n%s", output)
	}
	if !strings.Contains(output, "beta-tool") {
		t.Errorf("expected 'beta-tool' (write-granted) in output, got:\n%s", output)
	}
	if strings.Contains(output, "delta-tool") {
		t.Errorf("expected 'delta-tool' (granted to g2) NOT to appear, got:\n%s", output)
	}
	if strings.Contains(output, "Public") {
		t.Errorf("expected no Public section without --include-public, got:\n%s", output)
	}
}

func TestShowToolsCommand_PermissionFilter(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowToolsFlags()

	if showToolsCmd.Flags().Lookup("output") == nil {
		showToolsCmd.Flags().String("output", "", "")
	}
	showToolsCmd.Flags().Set("output", "")
	showToolsCmd.Flags().Set("permission", "write")

	buf := new(bytes.Buffer)
	showToolsCmd.SetOut(buf)

	err := showToolsCmd.RunE(showToolsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "── Write ──") {
		t.Errorf("expected Write section, got:\n%s", output)
	}
	if strings.Contains(output, "── Read ──") {
		t.Errorf("expected no Read section with --permission write, got:\n%s", output)
	}
	if !strings.Contains(output, "beta-tool") {
		t.Errorf("expected 'beta-tool' in output, got:\n%s", output)
	}
	if strings.Contains(output, "alpha-tool") {
		t.Errorf("expected 'alpha-tool' (read) NOT in output, got:\n%s", output)
	}
}

func TestShowToolsCommand_IncludePublic(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowToolsFlags()

	if showToolsCmd.Flags().Lookup("output") == nil {
		showToolsCmd.Flags().String("output", "", "")
	}
	showToolsCmd.Flags().Set("output", "")
	showToolsCmd.Flags().Set("permission", "all")
	showToolsCmd.Flags().Set("include-public", "true")

	buf := new(bytes.Buffer)
	showToolsCmd.SetOut(buf)

	err := showToolsCmd.RunE(showToolsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Public") {
		t.Errorf("expected Public section, got:\n%s", output)
	}
	if !strings.Contains(output, "gamma-tool") {
		t.Errorf("expected 'gamma-tool' (public) in output, got:\n%s", output)
	}
}

func TestShowToolsCommand_JSONOutput(t *testing.T) {
	server := newGroupsServer(t)
	defer server.Close()
	cleanup := setupTestConfig(t, server.URL)
	defer cleanup()
	defer resetShowToolsFlags()

	if showToolsCmd.Flags().Lookup("output") == nil {
		showToolsCmd.Flags().String("output", "", "")
	}
	showToolsCmd.Flags().Set("output", "json")
	showToolsCmd.Flags().Set("permission", "all")
	showToolsCmd.Flags().Set("include-public", "true")

	buf := new(bytes.Buffer)
	showToolsCmd.SetOut(buf)

	err := showToolsCmd.RunE(showToolsCmd, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Group struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"group"`
		Read   []api.Tool `json:"read"`
		Write  []api.Tool `json:"write"`
		Public []api.Tool `json:"public"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\nOutput:\n%s", err, buf.String())
	}
	if result.Group.ID != "g1" || result.Group.Name != "developers" {
		t.Errorf("unexpected group ref: %+v", result.Group)
	}
	if len(result.Read) != 1 || result.Read[0].ID != "t-alpha" {
		t.Errorf("expected one read tool 't-alpha', got: %+v", result.Read)
	}
	if len(result.Write) != 1 || result.Write[0].ID != "t-beta" {
		t.Errorf("expected one write tool 't-beta', got: %+v", result.Write)
	}
	if len(result.Public) != 1 || result.Public[0].ID != "t-gamma" {
		t.Errorf("expected one public tool 't-gamma', got: %+v", result.Public)
	}
}

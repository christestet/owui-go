package shared

import (
	"strings"
	"testing"

	"github.com/christestet/owui-go/internal/api"
)

func strptr(value string) *string { return &value }

func TestResolveUser(t *testing.T) {
	users := []api.User{
		{ID: "u1", Name: "Alex", Email: "alex.one@example.com", Username: strptr("alex"), Role: "user"},
		{ID: "u2", Name: "Alex", Email: "alex.two@example.com", Username: strptr("alex"), Role: "user"},
		{ID: "u3", Name: "Casey", Email: "casey@example.com", Username: strptr("casey"), Role: "user"},
	}
	tests := []struct {
		name       string
		identifier string
		wantID     string
		wantError  string
	}{
		{name: "exact ID wins", identifier: "u2", wantID: "u2"},
		{name: "email", identifier: "alex.one@example.com", wantID: "u1"},
		{name: "unique username", identifier: "casey", wantID: "u3"},
		{name: "unique name", identifier: "Casey", wantID: "u3"},
		{name: "duplicate display name", identifier: "Alex", wantError: "ambiguous"},
		{name: "duplicate username", identifier: "alex", wantError: "ambiguous"},
		{name: "missing", identifier: "nobody", wantError: "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := ResolveUser(users, tt.identifier)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("ResolveUser() error = %v, want error containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveUser() error = %v", err)
			}
			if user.ID != tt.wantID {
				t.Fatalf("ResolveUser() ID = %q, want %q", user.ID, tt.wantID)
			}
		})
	}
}

func TestResolveUsersDeduplicatesByID(t *testing.T) {
	users := []api.User{{ID: "u1", Name: "Alex", Email: "alex@example.com"}}
	resolved, err := ResolveUsers(users, []string{"Alex", "alex@example.com", "u1"})
	if err != nil {
		t.Fatalf("ResolveUsers() error = %v", err)
	}
	if len(resolved) != 1 || resolved[0].ID != "u1" {
		t.Fatalf("ResolveUsers() = %+v, want one u1", resolved)
	}
}

func TestResolveGroup(t *testing.T) {
	groups := []api.Group{
		{ID: "g1", Name: "team", Description: "First"},
		{ID: "g2", Name: "team", Description: "Second"},
		{ID: "g3", Name: "unique", Description: "Third"},
	}
	tests := []struct {
		name       string
		identifier string
		wantID     string
		wantError  string
	}{
		{name: "ID disambiguates", identifier: "g2", wantID: "g2"},
		{name: "unique name", identifier: "unique", wantID: "g3"},
		{name: "duplicate name", identifier: "team", wantError: "ambiguous"},
		{name: "missing", identifier: "missing", wantError: "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := ResolveGroup(groups, tt.identifier)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("ResolveGroup() error = %v, want error containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveGroup() error = %v", err)
			}
			if group.ID != tt.wantID {
				t.Fatalf("ResolveGroup() ID = %q, want %q", group.ID, tt.wantID)
			}
		})
	}
}

func TestInteractiveOptionsUseStableIDsAndHumanLabels(t *testing.T) {
	members := 3
	users := []api.User{
		{ID: "u1", Name: "Alex", Email: "one@example.com"},
		{ID: "u2", Name: "Alex", Email: "two@example.com"},
	}
	userOptions := UserOptions(users)
	for i, option := range userOptions {
		if option.Value != users[i].ID {
			t.Errorf("UserOptions()[%d].Value = %q, want %q", i, option.Value, users[i].ID)
		}
		if !strings.Contains(option.Key, users[i].Name) || !strings.Contains(option.Key, users[i].Email) {
			t.Errorf("UserOptions()[%d].Key = %q, want name and email", i, option.Key)
		}
	}

	groups := []api.Group{
		{ID: "g1", Name: "team", Description: "Same", MemberCount: &members},
		{ID: "g2", Name: "team", Description: "Same", MemberCount: &members},
		{ID: "g3", Name: "team", Description: "Different", MemberCount: &members},
	}
	groupOptions := GroupOptions(groups)
	for i, option := range groupOptions {
		if option.Value != groups[i].ID {
			t.Errorf("GroupOptions()[%d].Value = %q, want %q", i, option.Value, groups[i].ID)
		}
		if !strings.Contains(option.Key, groups[i].Name) || !strings.Contains(option.Key, groups[i].Description) {
			t.Errorf("GroupOptions()[%d].Key = %q, want human metadata", i, option.Key)
		}
	}
	if !strings.Contains(groupOptions[0].Key, "g1") || !strings.Contains(groupOptions[1].Key, "g2") {
		t.Errorf("identical group metadata must show IDs: %q, %q", groupOptions[0].Key, groupOptions[1].Key)
	}
	if strings.Contains(groupOptions[2].Key, "g3") {
		t.Errorf("distinct human metadata should not expose ID: %q", groupOptions[2].Key)
	}
}

func TestCompletionsAreUnambiguous(t *testing.T) {
	users := []api.User{
		{ID: "u1", Name: "Alex", Email: "one@example.com"},
		{ID: "u2", Name: "Alex", Email: "two@example.com"},
	}
	userCompletions := UserCompletions(users, nil, "Alex")
	if len(userCompletions) != 2 || !strings.HasPrefix(userCompletions[0], "one@example.com\t") || !strings.HasPrefix(userCompletions[1], "two@example.com\t") {
		t.Fatalf("UserCompletions() = %v, want unique email insertions", userCompletions)
	}

	groups := []api.Group{{ID: "g1", Name: "team"}, {ID: "g2", Name: "team"}, {ID: "g3", Name: "other"}}
	groupCompletions := GroupCompletions(groups, nil, "team")
	if len(groupCompletions) != 2 || !strings.HasPrefix(groupCompletions[0], "g1\t") || !strings.HasPrefix(groupCompletions[1], "g2\t") {
		t.Fatalf("GroupCompletions() = %v, want IDs for duplicate names", groupCompletions)
	}
}

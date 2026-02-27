package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestListUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/users/" {
			t.Errorf("expected path /api/v1/users/, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UserListResponse{
			Users: []User{
				{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "admin"},
				{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	users, err := client.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "alice" {
		t.Errorf("expected first user name 'alice', got %q", users[0].Name)
	}
	if users[1].Role != "user" {
		t.Errorf("expected second user role 'user', got %q", users[1].Role)
	}
}

func TestListUsers_Pagination(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			json.NewEncoder(w).Encode(UserListResponse{
				Users: []User{
					{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "admin"},
					{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
				},
				Total: 3,
			})
		case "2":
			json.NewEncoder(w).Encode(UserListResponse{
				Users: []User{
					{ID: "u3", Name: "charlie", Email: "charlie@example.com", Role: "user"},
				},
				Total: 3,
			})
		default:
			json.NewEncoder(w).Encode(UserListResponse{Users: []User{}, Total: 3})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	users, err := client.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users across pages, got %d", len(users))
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests (page 1 + page 2 in parallel), got %d", requestCount)
	}
}

func TestListUsersWithOptions_Query(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("query")
		json.NewEncoder(w).Encode(UserListResponse{
			Users: []User{
				{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "admin"},
			},
			Total: 1,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	users, err := client.ListUsersWithOptions(context.Background(), &UserListOptions{Query: "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if receivedQuery != "alice" {
		t.Errorf("expected query param 'alice', got %q", receivedQuery)
	}
}

func TestListUsers_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":"unauthorized"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key", 1)
	_, err := client.ListUsers(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateUser(t *testing.T) {
	var receivedForm CreateUserForm

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/auths/add" {
			t.Errorf("expected path /api/v1/auths/add, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedForm)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"abc"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	form := CreateUserForm{
		Name:     "charlie",
		Email:    "charlie@example.com",
		Password: "pass123",
		Role:     "user",
	}
	err := client.CreateUser(context.Background(), form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedForm.Name != "charlie" {
		t.Errorf("expected name 'charlie', got %q", receivedForm.Name)
	}
	if receivedForm.Email != "charlie@example.com" {
		t.Errorf("expected email 'charlie@example.com', got %q", receivedForm.Email)
	}
	if receivedForm.Role != "user" {
		t.Errorf("expected role 'user', got %q", receivedForm.Role)
	}
}

func TestCreateUser_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":"email already exists"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.CreateUser(context.Background(), CreateUserForm{
		Name: "dup", Email: "dup@example.com", Password: "x", Role: "user",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/users/u1" {
			t.Errorf("expected path /api/v1/users/u1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`true`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.DeleteUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUser_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.DeleteUser(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateUser(t *testing.T) {
	var receivedForm UpdateUserForm

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/users/u1/update" {
			t.Errorf("expected path /api/v1/users/u1/update, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedForm)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"u1","role":"admin"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	form := UpdateUserForm{
		Role:  "admin",
		Name:  "alice",
		Email: "alice@example.com",
	}
	err := client.UpdateUser(context.Background(), "u1", form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedForm.Role != "admin" {
		t.Errorf("expected role 'admin', got %q", receivedForm.Role)
	}
}

func TestListGroups(t *testing.T) {
	memberCount := 3
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/" {
			t.Errorf("expected path /api/v1/groups/, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]Group{
			{ID: "g1", Name: "developers", Description: "Dev team", MemberCount: &memberCount},
			{ID: "g2", Name: "oauth-group", Description: "Group oauth-group created automatically via OAuth."},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	groups, err := client.ListGroups(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "developers" {
		t.Errorf("expected group name 'developers', got %q", groups[0].Name)
	}
}

func TestListGroups_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"detail":"forbidden"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	_, err := client.ListGroups(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExportGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/id/g1/export" {
			t.Errorf("expected path /api/v1/groups/id/g1/export, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Group{
			ID:      "g1",
			Name:    "developers",
			UserIDs: []string{"u1", "u2"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	group, err := client.ExportGroup(context.Background(), "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "developers" {
		t.Errorf("expected group name 'developers', got %q", group.Name)
	}
	if len(group.UserIDs) != 2 {
		t.Errorf("expected 2 user IDs, got %d", len(group.UserIDs))
	}
}

func TestAddUsersToGroup(t *testing.T) {
	var receivedForm UserIdsForm

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/id/g1/users/add" {
			t.Errorf("expected path /api/v1/groups/id/g1/users/add, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedForm)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"g1","name":"developers"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.AddUsersToGroup(context.Background(), "g1", []string{"u1", "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(receivedForm.UserIDs) != 2 {
		t.Errorf("expected 2 user IDs in request, got %d", len(receivedForm.UserIDs))
	}
}

func TestAddUsersToGroup_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"group not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.AddUsersToGroup(context.Background(), "nonexistent", []string{"u1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveUsersFromGroup(t *testing.T) {
	var receivedForm UserIdsForm

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/id/g1/users/remove" {
			t.Errorf("expected path /api/v1/groups/id/g1/users/remove, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedForm)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"g1","name":"developers"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.RemoveUsersFromGroup(context.Background(), "g1", []string{"u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(receivedForm.UserIDs) != 1 {
		t.Errorf("expected 1 user ID in request, got %d", len(receivedForm.UserIDs))
	}
}

func TestRemoveUsersFromGroup_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"server error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.RemoveUsersFromGroup(context.Background(), "g1", []string{"u1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

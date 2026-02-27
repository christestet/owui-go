package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateGroup(t *testing.T) {
	var receivedForm GroupForm

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/create" {
			t.Errorf("expected path /api/v1/groups/create, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedForm)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Group{
			ID:          "g-new",
			Name:        receivedForm.Name,
			Description: receivedForm.Description,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	form := GroupForm{
		Name:        "backend-team",
		Description: "Backend development team",
	}
	group, err := client.CreateGroup(context.Background(), form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.ID != "g-new" {
		t.Errorf("expected ID 'g-new', got %q", group.ID)
	}
	if receivedForm.Name != "backend-team" {
		t.Errorf("expected name 'backend-team', got %q", receivedForm.Name)
	}
	if receivedForm.Description != "Backend development team" {
		t.Errorf("expected description 'Backend development team', got %q", receivedForm.Description)
	}
}

func TestCreateGroup_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":"group already exists"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	_, err := client.CreateGroup(context.Background(), GroupForm{Name: "dup", Description: "dup"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateGroup_WithPermissions(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Group{ID: "g-new", Name: "team"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	form := GroupForm{
		Name:        "team",
		Description: "My team",
		Permissions: json.RawMessage(`{"workspace":{"models":true}}`),
	}
	_, err := client.CreateGroup(context.Background(), form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	perms, ok := receivedBody["permissions"]
	if !ok {
		t.Fatal("expected permissions in request body")
	}
	permsMap, ok := perms.(map[string]interface{})
	if !ok {
		t.Fatalf("expected permissions to be object, got %T", perms)
	}
	if _, ok := permsMap["workspace"]; !ok {
		t.Error("expected 'workspace' key in permissions")
	}
}

func TestDeleteGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/id/g1/delete" {
			t.Errorf("expected path /api/v1/groups/id/g1/delete, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`true`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.DeleteGroup(context.Background(), "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteGroup_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"group not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	err := client.DeleteGroup(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateGroup(t *testing.T) {
	var receivedForm GroupUpdateForm

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/id/g1/update" {
			t.Errorf("expected path /api/v1/groups/id/g1/update, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedForm)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Group{
			ID:          "g1",
			Name:        receivedForm.Name,
			Description: receivedForm.Description,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	form := GroupUpdateForm{
		Name:        "new-name",
		Description: "new description",
	}
	group, err := client.UpdateGroup(context.Background(), "g1", form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "new-name" {
		t.Errorf("expected name 'new-name', got %q", group.Name)
	}
	if receivedForm.Description != "new description" {
		t.Errorf("expected description 'new description', got %q", receivedForm.Description)
	}
}

func TestUpdateGroup_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"group not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	_, err := client.UpdateGroup(context.Background(), "nonexistent", GroupUpdateForm{Name: "x", Description: "y"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetGroup(t *testing.T) {
	memberCount := 5
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/id/g1" {
			t.Errorf("expected path /api/v1/groups/id/g1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Group{
			ID:          "g1",
			Name:        "developers",
			Description: "Dev team",
			MemberCount: &memberCount,
			CreatedAt:   1700000000,
			UpdatedAt:   1700100000,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	group, err := client.GetGroup(context.Background(), "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "developers" {
		t.Errorf("expected name 'developers', got %q", group.Name)
	}
	if group.MemberCount == nil || *group.MemberCount != 5 {
		t.Errorf("expected member count 5, got %v", group.MemberCount)
	}
	if group.CreatedAt != 1700000000 {
		t.Errorf("expected created_at 1700000000, got %d", group.CreatedAt)
	}
}

func TestGetGroup_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	_, err := client.GetGroup(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetGroupMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/groups/id/g1/users" {
			t.Errorf("expected path /api/v1/groups/id/g1/users, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]User{
			{ID: "u1", Name: "alice", Email: "alice@example.com", Role: "user"},
			{ID: "u2", Name: "bob", Email: "bob@example.com", Role: "user"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	members, err := client.GetGroupMembers(context.Background(), "g1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].Name != "alice" {
		t.Errorf("expected first member 'alice', got %q", members[0].Name)
	}
}

func TestGetGroupMembers_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"group not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	_, err := client.GetGroupMembers(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

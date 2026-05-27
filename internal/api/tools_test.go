package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/tools/list" {
			t.Errorf("expected path /api/v1/tools/list, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]Tool{
			{
				ID: "weather", UserID: "u1", Name: "Weather",
				AccessGrants: []AccessGrantModel{
					{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
				},
			},
			{ID: "calc", UserID: "u1", Name: "Calculator", AccessGrants: []AccessGrantModel{}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if len(tools[0].AccessGrants) != 1 || tools[0].AccessGrants[0].PrincipalID != "g1" {
		t.Errorf("expected grant for g1 on first tool, got %+v", tools[0].AccessGrants)
	}
	if len(tools[1].AccessGrants) != 0 {
		t.Errorf("expected second tool public (no grants), got %d", len(tools[1].AccessGrants))
	}
}

func TestListTools_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"detail":"forbidden"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

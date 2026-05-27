package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListFunctions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/functions/" {
			t.Errorf("expected path /api/v1/functions/, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		desc := "filters out emojis"
		json.NewEncoder(w).Encode([]Function{
			{
				ID:       "emoji_filter",
				UserID:   "u1",
				Name:     "Emoji Filter",
				Type:     "filter",
				Meta:     FunctionMeta{Description: &desc},
				IsActive: true,
				IsGlobal: true,
			},
			{
				ID:       "private_action",
				UserID:   "u1",
				Name:     "Private Action",
				Type:     "action",
				IsActive: false,
				IsGlobal: false,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	fns, err := client.ListFunctions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fns) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(fns))
	}
	if fns[0].Type != "filter" {
		t.Errorf("expected first type 'filter', got %q", fns[0].Type)
	}
	if !fns[0].IsActive || !fns[0].IsGlobal {
		t.Errorf("expected first function active+global, got active=%v global=%v", fns[0].IsActive, fns[0].IsGlobal)
	}
	if fns[1].IsActive {
		t.Errorf("expected second function inactive, got active=%v", fns[1].IsActive)
	}
}

func TestListFunctions_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"detail":"forbidden"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 1)
	if _, err := client.ListFunctions(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

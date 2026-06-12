package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newModelsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/models/list":
			page := r.URL.Query().Get("page")
			if page == "" || page == "1" {
				json.NewEncoder(w).Encode(ModelAccessListResponse{
					Items: []ModelAccessResponse{
						{ID: "gpt-4o", Name: "GPT-4o", BaseModelID: "openai/gpt-4o", IsActive: true, AccessGrants: []AccessGrantModel{}},
						{ID: "claude-sonnet", Name: "Claude Sonnet", BaseModelID: "anthropic/claude-3.5", IsActive: true, AccessGrants: []AccessGrantModel{{PrincipalType: "group", PrincipalID: "g1", Permission: "read"}}},
					},
					Total: 2,
				})
			} else {
				json.NewEncoder(w).Encode(ModelAccessListResponse{Items: []ModelAccessResponse{}, Total: 2})
			}
		case r.Method == "GET" && r.URL.Path == "/api/v1/models/model":
			id := r.URL.Query().Get("id")
			if id == "gpt-4o" {
				json.NewEncoder(w).Encode(ModelAccessResponse{
					ID:           "gpt-4o",
					Name:         "GPT-4o",
					BaseModelID:  "openai/gpt-4o",
					IsActive:     true,
					AccessGrants: []AccessGrantModel{},
					Meta:         ModelMeta{Description: "GPT-4o model"},
					User:         &ModelUser{ID: "u1", Name: "admin", Email: "admin@example.com"},
				})
			} else {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"detail":"model not found"}`))
			}
		case r.Method == "POST" && r.URL.Path == "/api/v1/models/model/toggle":
			json.NewEncoder(w).Encode(ModelResponse{
				ID:       "gpt-4o",
				Name:     "GPT-4o",
				IsActive: false,
			})
		case r.Method == "POST" && r.URL.Path == "/api/v1/models/model/access/update":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"detail":"not found"}`))
		}
	}))
}

func TestListModels(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	models, err := client.ListModels(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "gpt-4o" {
		t.Errorf("expected first model ID 'gpt-4o', got %q", models[0].ID)
	}
	if models[1].ID != "claude-sonnet" {
		t.Errorf("expected second model ID 'claude-sonnet', got %q", models[1].ID)
	}
}

func TestListModelsWithOptions_Query(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	models, err := client.ListModelsWithOptions(t.Context(), &ModelListOptions{Query: "gpt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Server doesn't actually filter in test, just check it doesn't error
	if len(models) == 0 {
		t.Error("expected models in response")
	}
}

func TestListModels_Pagination(t *testing.T) {
	// Server that returns 2 pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch page {
		case "", "1":
			json.NewEncoder(w).Encode(ModelAccessListResponse{
				Items: []ModelAccessResponse{
					{ID: "model-1", Name: "Model 1"},
				},
				Total: 2,
			})
		case "2":
			json.NewEncoder(w).Encode(ModelAccessListResponse{
				Items: []ModelAccessResponse{
					{ID: "model-2", Name: "Model 2"},
				},
				Total: 2,
			})
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	models, err := client.ListModels(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models from 2 pages, got %d", len(models))
	}
	if models[0].ID != "model-1" || models[1].ID != "model-2" {
		t.Errorf("unexpected model order: %v, %v", models[0].ID, models[1].ID)
	}
}

func TestListModels_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ModelAccessListResponse{Items: []ModelAccessResponse{}, Total: 0})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	models, err := client.ListModels(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("expected 0 models, got %d", len(models))
	}
}

func TestListModels_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"server error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	_, err := client.ListModels(t.Context())
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

func TestGetModel(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	model, err := client.GetModel(t.Context(), "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.ID != "gpt-4o" {
		t.Errorf("expected ID 'gpt-4o', got %q", model.ID)
	}
	if model.Name != "GPT-4o" {
		t.Errorf("expected Name 'GPT-4o', got %q", model.Name)
	}
	if model.Meta.Description != "GPT-4o model" {
		t.Errorf("expected Description 'GPT-4o model', got %q", model.Meta.Description)
	}
}

func TestGetModel_NotFound(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	_, err := client.GetModel(t.Context(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestToggleModel(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	resp, err := client.ToggleModel(t.Context(), "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsActive {
		t.Error("expected model to be toggled to inactive")
	}
}

func TestUpdateModelAccess(t *testing.T) {
	server := newModelsServer(t)
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5)
	form := ModelAccessGrantsForm{
		ID: "gpt-4o",
		AccessGrants: []AccessGrantModel{
			{PrincipalType: "group", PrincipalID: "g1", Permission: "read"},
		},
	}
	err := client.UpdateModelAccess(t.Context(), form)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

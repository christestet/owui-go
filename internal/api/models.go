package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
)

// ModelCapabilities describes which capabilities a model supports.
type ModelCapabilities struct {
	Vision          *bool `json:"vision,omitempty"`
	Citations       *bool `json:"citations,omitempty"`
	CodeInterpreter *bool `json:"code_interpreter,omitempty"`
}

// ModelMeta contains model metadata.
type ModelMeta struct {
	ProfileImageURL string            `json:"profile_image_url,omitempty"`
	Description     string            `json:"description,omitempty"`
	Capabilities    ModelCapabilities  `json:"capabilities,omitempty"`
	Tags            []ModelTag        `json:"tags,omitempty"`
}

// ModelTag represents a tag on a model.
type ModelTag struct {
	Name string `json:"name"`
}

// ModelUser represents the owner of a model.
type ModelUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AccessGrantModel represents a single access grant entry for a model.
type AccessGrantModel struct {
	ID            string `json:"id,omitempty"`
	ResourceType  string `json:"resource_type,omitempty"`
	ResourceID    string `json:"resource_id,omitempty"`
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	Permission    string `json:"permission"`
	CreatedAt     int64  `json:"created_at,omitempty"`
}

// ModelAccessResponse represents a single model from the list/detail endpoints.
type ModelAccessResponse struct {
	ID           string             `json:"id"`
	UserID       string             `json:"user_id,omitempty"`
	BaseModelID  string             `json:"base_model_id,omitempty"`
	Name         string             `json:"name"`
	Params       json.RawMessage    `json:"params,omitempty"`
	Meta         ModelMeta          `json:"meta"`
	AccessGrants []AccessGrantModel `json:"access_grants"`
	IsActive     bool               `json:"is_active"`
	UpdatedAt    int64              `json:"updated_at,omitempty"`
	CreatedAt    int64              `json:"created_at,omitempty"`
	User         *ModelUser         `json:"user,omitempty"`
	WriteAccess  *bool              `json:"write_access,omitempty"`
}

// ModelAccessListResponse is the paginated response from the models list endpoint.
type ModelAccessListResponse struct {
	Items []ModelAccessResponse `json:"items"`
	Total int                   `json:"total"`
}

// ModelAccessGrantsForm is the request body for updating model access grants.
type ModelAccessGrantsForm struct {
	ID           string             `json:"id"`
	Name         string             `json:"name,omitempty"`
	AccessGrants []AccessGrantModel `json:"access_grants"`
}

// ModelResponse is the response from the toggle endpoint.
type ModelResponse struct {
	ID           string             `json:"id"`
	UserID       string             `json:"user_id,omitempty"`
	BaseModelID  string             `json:"base_model_id,omitempty"`
	Name         string             `json:"name"`
	Params       json.RawMessage    `json:"params,omitempty"`
	Meta         json.RawMessage    `json:"meta,omitempty"`
	AccessGrants []AccessGrantModel `json:"access_grants"`
	IsActive     bool               `json:"is_active"`
	UpdatedAt    int64              `json:"updated_at,omitempty"`
	CreatedAt    int64              `json:"created_at,omitempty"`
}

// ModelListOptions configures the model list request.
type ModelListOptions struct {
	Query string // server-side search by model name/id
	Tag   string // filter by tag
}

// ListModelsWithOptions fetches all models, auto-paginating through all pages.
// Page 1 is fetched first to determine the total; remaining pages are fetched
// in parallel using goroutines. If opts is nil, no filtering is applied.
func (c *Client) ListModelsWithOptions(ctx context.Context, opts *ModelListOptions) ([]ModelAccessResponse, error) {
	const maxPages = 100

	fetchPage := func(page int) (ModelAccessListResponse, error) {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		if opts != nil && opts.Query != "" {
			params.Set("query", opts.Query)
		}
		if opts != nil && opts.Tag != "" {
			params.Set("tag", opts.Tag)
		}
		endpoint := "/api/v1/models/list?" + params.Encode()
		body, err := c.sendRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return ModelAccessListResponse{}, fmt.Errorf("listing models (page %d): %w", page, err)
		}
		var resp ModelAccessListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return ModelAccessListResponse{}, fmt.Errorf("decoding models response: %w", err)
		}
		return resp, nil
	}

	// Fetch page 1 to learn the total and page size.
	first, err := fetchPage(1)
	if err != nil {
		return nil, err
	}
	if len(first.Items) == 0 || len(first.Items) >= first.Total {
		return first.Items, nil
	}

	pageSize := len(first.Items)
	totalPages := (first.Total + pageSize - 1) / pageSize
	if totalPages > maxPages {
		totalPages = maxPages
	}

	// Fetch remaining pages in parallel.
	type pageResult struct {
		page   int
		models []ModelAccessResponse
		err    error
	}
	results := make([]pageResult, totalPages-1)
	var wg sync.WaitGroup
	for i := 2; i <= totalPages; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			resp, err := fetchPage(page)
			results[page-2] = pageResult{page: page, models: resp.Items, err: err}
		}(i)
	}
	wg.Wait()

	allModels := make([]ModelAccessResponse, 0, first.Total)
	allModels = append(allModels, first.Items...)
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		allModels = append(allModels, r.models...)
	}

	return allModels, nil
}

// ListModels fetches all models from the Open WebUI instance (auto-paginates).
func (c *Client) ListModels(ctx context.Context) ([]ModelAccessResponse, error) {
	return c.ListModelsWithOptions(ctx, nil)
}

// GetModel fetches a single model by its ID.
func (c *Client) GetModel(ctx context.Context, modelID string) (*ModelAccessResponse, error) {
	params := url.Values{}
	params.Set("id", modelID)
	endpoint := "/api/v1/models/model?" + params.Encode()
	body, err := c.sendRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting model: %w", err)
	}
	var model ModelAccessResponse
	if err := json.Unmarshal(body, &model); err != nil {
		return nil, fmt.Errorf("decoding model response: %w", err)
	}
	return &model, nil
}

// ToggleModel toggles the is_active state of a model.
func (c *Client) ToggleModel(ctx context.Context, modelID string) (*ModelResponse, error) {
	params := url.Values{}
	params.Set("id", modelID)
	endpoint := "/api/v1/models/model/toggle?" + params.Encode()
	body, err := c.sendRequest(ctx, "POST", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("toggling model: %w", err)
	}
	var resp ModelResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding toggle response: %w", err)
	}
	return &resp, nil
}

// UpdateModelAccess updates the access grants for a model.
func (c *Client) UpdateModelAccess(ctx context.Context, form ModelAccessGrantsForm) error {
	_, err := c.sendRequest(ctx, "POST", "/api/v1/models/model/access/update", form)
	if err != nil {
		return fmt.Errorf("updating model access: %w", err)
	}
	return nil
}

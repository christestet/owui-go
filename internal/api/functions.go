package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// FunctionMeta holds the user-facing metadata of a function.
type FunctionMeta struct {
	Description *string         `json:"description,omitempty"`
	Manifest    json.RawMessage `json:"manifest,omitempty"`
}

// Function represents a function in Open WebUI as returned by the list/detail endpoints.
// Note: the API has no per-group access control for functions; visibility is binary
// (is_global=true → all users; is_global=false → owner-only).
type Function struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Meta      FunctionMeta `json:"meta"`
	IsActive  bool         `json:"is_active"`
	IsGlobal  bool         `json:"is_global"`
	CreatedAt int64        `json:"created_at"`
	UpdatedAt int64        `json:"updated_at"`
}

// ListFunctions fetches all functions visible to the current user.
func (c *Client) ListFunctions(ctx context.Context) ([]Function, error) {
	body, err := c.sendRequest(ctx, "GET", "/api/v1/functions/", nil)
	if err != nil {
		return nil, fmt.Errorf("listing functions: %w", err)
	}
	var functions []Function
	if err := json.Unmarshal(body, &functions); err != nil {
		return nil, fmt.Errorf("decoding functions response: %w", err)
	}
	return functions, nil
}

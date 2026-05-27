package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolMeta holds the user-facing metadata of a tool.
type ToolMeta struct {
	Description *string         `json:"description,omitempty"`
	Manifest    json.RawMessage `json:"manifest,omitempty"`
}

// Tool represents a tool as returned by /api/v1/tools/list (ToolAccessResponse).
// Tools support per-group/per-user RBAC via access_grants (unlike functions, which
// only have a global/private toggle).
type Tool struct {
	ID           string             `json:"id"`
	UserID       string             `json:"user_id,omitempty"`
	Name         string             `json:"name"`
	Meta         ToolMeta           `json:"meta"`
	AccessGrants []AccessGrantModel `json:"access_grants"`
	User         *ModelUser         `json:"user,omitempty"`
	WriteAccess  *bool              `json:"write_access,omitempty"`
	UpdatedAt    int64              `json:"updated_at,omitempty"`
	CreatedAt    int64              `json:"created_at,omitempty"`
}

// ListTools fetches all tools visible to the current user, including access grants.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	body, err := c.sendRequest(ctx, "GET", "/api/v1/tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("listing tools: %w", err)
	}
	var tools []Tool
	if err := json.Unmarshal(body, &tools); err != nil {
		return nil, fmt.Errorf("decoding tools response: %w", err)
	}
	return tools, nil
}

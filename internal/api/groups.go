package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// GroupForm is the request body for creating a new group.
type GroupForm struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Permissions json.RawMessage `json:"permissions,omitempty"`
}

// GroupUpdateForm is the request body for updating a group.
type GroupUpdateForm struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Permissions json.RawMessage `json:"permissions,omitempty"`
}

// CreateGroup creates a new group in the Open WebUI instance.
func (c *Client) CreateGroup(ctx context.Context, form GroupForm) (*Group, error) {
	body, err := c.sendRequest(ctx, "POST", "/api/v1/groups/create", form)
	if err != nil {
		return nil, fmt.Errorf("creating group: %w", err)
	}
	var group Group
	if err := json.Unmarshal(body, &group); err != nil {
		return nil, fmt.Errorf("decoding group response: %w", err)
	}
	return &group, nil
}

// DeleteGroup deletes a group by ID.
func (c *Client) DeleteGroup(ctx context.Context, groupID string) error {
	_, err := c.sendRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/groups/id/%s/delete", groupID), nil)
	if err != nil {
		return fmt.Errorf("deleting group: %w", err)
	}
	return nil
}

// UpdateGroup updates a group by ID.
func (c *Client) UpdateGroup(ctx context.Context, groupID string, form GroupUpdateForm) (*Group, error) {
	body, err := c.sendRequest(ctx, "POST", fmt.Sprintf("/api/v1/groups/id/%s/update", groupID), form)
	if err != nil {
		return nil, fmt.Errorf("updating group: %w", err)
	}
	var group Group
	if err := json.Unmarshal(body, &group); err != nil {
		return nil, fmt.Errorf("decoding group response: %w", err)
	}
	return &group, nil
}

// GetGroup fetches a single group by ID.
func (c *Client) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	body, err := c.sendRequest(ctx, "GET", fmt.Sprintf("/api/v1/groups/id/%s", groupID), nil)
	if err != nil {
		return nil, fmt.Errorf("getting group: %w", err)
	}
	var group Group
	if err := json.Unmarshal(body, &group); err != nil {
		return nil, fmt.Errorf("decoding group response: %w", err)
	}
	return &group, nil
}

// GetGroupMembers fetches the members of a group.
func (c *Client) GetGroupMembers(ctx context.Context, groupID string) ([]User, error) {
	body, err := c.sendRequest(ctx, "POST", fmt.Sprintf("/api/v1/groups/id/%s/users", groupID), nil)
	if err != nil {
		return nil, fmt.Errorf("getting group members: %w", err)
	}
	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("decoding group members response: %w", err)
	}
	return users, nil
}

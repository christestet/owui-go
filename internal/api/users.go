package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
)

// User represents a user in Open WebUI.
type User struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Username *string `json:"username"`
	Role     string  `json:"role"`
}

// UserListResponse is the response from the list users endpoint.
type UserListResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

// CreateUserForm is the request body for creating a new user.
type CreateUserForm struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateUserForm is the request body for updating a user.
type UpdateUserForm struct {
	Role            string `json:"role"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	ProfileImageURL string `json:"profile_image_url"`
}

// Group represents a group in Open WebUI.
type Group struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	UserIDs     []string `json:"user_ids"`
	MemberCount *int     `json:"member_count"`
}

// UserIdsForm is the request body for adding/removing users from a group.
type UserIdsForm struct {
	UserIDs []string `json:"user_ids"`
}

// UserListOptions configures the user list request.
type UserListOptions struct {
	Query string // server-side search filter
}

// ListUsersWithOptions fetches all users, auto-paginating through all pages.
// Page 1 is fetched first to determine the total; remaining pages are fetched
// in parallel using goroutines. If opts is nil, no filtering is applied.
func (c *Client) ListUsersWithOptions(ctx context.Context, opts *UserListOptions) ([]User, error) {
	const maxPages = 100

	fetchPage := func(page int) (UserListResponse, error) {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		if opts != nil && opts.Query != "" {
			params.Set("query", opts.Query)
		}
		endpoint := "/api/v1/users/?" + params.Encode()
		body, err := c.sendRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return UserListResponse{}, fmt.Errorf("listing users (page %d): %w", page, err)
		}
		var resp UserListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return UserListResponse{}, fmt.Errorf("decoding users response: %w", err)
		}
		return resp, nil
	}

	// Fetch page 1 to learn the total and page size.
	first, err := fetchPage(1)
	if err != nil {
		return nil, err
	}
	if len(first.Users) == 0 || len(first.Users) >= first.Total {
		return first.Users, nil
	}

	pageSize := len(first.Users)
	totalPages := (first.Total + pageSize - 1) / pageSize
	if totalPages > maxPages {
		totalPages = maxPages
	}

	// Fetch remaining pages in parallel.
	type pageResult struct {
		page  int
		users []User
		err   error
	}
	results := make([]pageResult, totalPages-1)
	var wg sync.WaitGroup
	for i := 2; i <= totalPages; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			resp, err := fetchPage(page)
			results[page-2] = pageResult{page: page, users: resp.Users, err: err}
		}(i)
	}
	wg.Wait()

	allUsers := make([]User, 0, first.Total)
	allUsers = append(allUsers, first.Users...)
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		allUsers = append(allUsers, r.users...)
	}

	return allUsers, nil
}

// ListUsers fetches all users from the Open WebUI instance (auto-paginates).
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	return c.ListUsersWithOptions(ctx, nil)
}

// CreateUser creates a new user in the Open WebUI instance.
func (c *Client) CreateUser(ctx context.Context, form CreateUserForm) error {
	_, err := c.sendRequest(ctx, "POST", "/api/v1/auths/add", form)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// DeleteUser deletes a user by ID.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	_, err := c.sendRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/users/%s", userID), nil)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

// UpdateUser updates a user by ID.
func (c *Client) UpdateUser(ctx context.Context, userID string, form UpdateUserForm) error {
	_, err := c.sendRequest(ctx, "POST", fmt.Sprintf("/api/v1/users/%s/update", userID), form)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

// ListGroups fetches all groups from the Open WebUI instance.
func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	body, err := c.sendRequest(ctx, "GET", "/api/v1/groups/", nil)
	if err != nil {
		return nil, fmt.Errorf("listing groups: %w", err)
	}
	var groups []Group
	if err := json.Unmarshal(body, &groups); err != nil {
		return nil, fmt.Errorf("decoding groups response: %w", err)
	}
	return groups, nil
}

// ExportGroup fetches a group by ID including its user_ids.
func (c *Client) ExportGroup(ctx context.Context, groupID string) (*Group, error) {
	body, err := c.sendRequest(ctx, "GET", fmt.Sprintf("/api/v1/groups/id/%s/export", groupID), nil)
	if err != nil {
		return nil, fmt.Errorf("exporting group: %w", err)
	}
	var group Group
	if err := json.Unmarshal(body, &group); err != nil {
		return nil, fmt.Errorf("decoding group response: %w", err)
	}
	return &group, nil
}

// AddUsersToGroup adds one or more users to a group.
func (c *Client) AddUsersToGroup(ctx context.Context, groupID string, userIDs []string) error {
	form := UserIdsForm{UserIDs: userIDs}
	_, err := c.sendRequest(ctx, "POST", fmt.Sprintf("/api/v1/groups/id/%s/users/add", groupID), form)
	if err != nil {
		return fmt.Errorf("adding users to group: %w", err)
	}
	return nil
}

// RemoveUsersFromGroup removes one or more users from a group.
func (c *Client) RemoveUsersFromGroup(ctx context.Context, groupID string, userIDs []string) error {
	form := UserIdsForm{UserIDs: userIDs}
	_, err := c.sendRequest(ctx, "POST", fmt.Sprintf("/api/v1/groups/id/%s/users/remove", groupID), form)
	if err != nil {
		return fmt.Errorf("removing users from group: %w", err)
	}
	return nil
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

// AddPipelineForm is the request body for adding a pipeline registration.
type AddPipelineForm struct {
	URL    string `json:"url"`
	URLIdx int    `json:"urlIdx"`
}

// DeletePipelineForm is the request body for removing a pipeline registration.
type DeletePipelineForm struct {
	ID     string `json:"id"`
	URLIdx int    `json:"urlIdx"`
}

func decodeUntypedJSON(body []byte) (any, error) {
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding untyped response: %w", err)
	}
	return out, nil
}

// ListPipelineRegistrationsRaw returns the untyped payload from /api/v1/pipelines/list.
func (c *Client) ListPipelineRegistrationsRaw(ctx context.Context) (any, error) {
	body, err := c.sendRequest(ctx, "GET", "/api/v1/pipelines/list", nil)
	if err != nil {
		return nil, fmt.Errorf("listing pipeline registrations: %w", err)
	}
	return decodeUntypedJSON(body)
}

// ListPipesByURLIdxRaw returns the untyped payload from /api/v1/pipelines/?urlIdx=<n>.
func (c *Client) ListPipesByURLIdxRaw(ctx context.Context, urlIdx int) (any, error) {
	params := url.Values{}
	params.Set("urlIdx", strconv.Itoa(urlIdx))
	endpoint := "/api/v1/pipelines/?" + params.Encode()
	body, err := c.sendRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("listing pipes for urlIdx %d: %w", urlIdx, err)
	}
	return decodeUntypedJSON(body)
}

// GetPipelineValvesRaw returns the untyped payload from
// /api/v1/pipelines/{pipeline_id}/valves?urlIdx=<n>.
func (c *Client) GetPipelineValvesRaw(ctx context.Context, pipelineID string, urlIdx int) (any, error) {
	params := url.Values{}
	params.Set("urlIdx", strconv.Itoa(urlIdx))
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/valves?%s", url.PathEscape(pipelineID), params.Encode())
	body, err := c.sendRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting valves for pipe %q (urlIdx=%d): %w", pipelineID, urlIdx, err)
	}
	return decodeUntypedJSON(body)
}

// GetPipelineValvesSpecRaw returns the untyped payload from
// /api/v1/pipelines/{pipeline_id}/valves/spec?urlIdx=<n>.
func (c *Client) GetPipelineValvesSpecRaw(ctx context.Context, pipelineID string, urlIdx int) (any, error) {
	params := url.Values{}
	params.Set("urlIdx", strconv.Itoa(urlIdx))
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/valves/spec?%s", url.PathEscape(pipelineID), params.Encode())
	body, err := c.sendRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting valves spec for pipe %q (urlIdx=%d): %w", pipelineID, urlIdx, err)
	}
	return decodeUntypedJSON(body)
}

// UpdatePipelineValvesRaw posts an arbitrary JSON object to
// /api/v1/pipelines/{pipeline_id}/valves/update?urlIdx=<n>.
func (c *Client) UpdatePipelineValvesRaw(ctx context.Context, pipelineID string, urlIdx int, data map[string]any) (any, error) {
	params := url.Values{}
	params.Set("urlIdx", strconv.Itoa(urlIdx))
	endpoint := fmt.Sprintf("/api/v1/pipelines/%s/valves/update?%s", url.PathEscape(pipelineID), params.Encode())
	body, err := c.sendRequest(ctx, "POST", endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("updating valves for pipe %q (urlIdx=%d): %w", pipelineID, urlIdx, err)
	}
	return decodeUntypedJSON(body)
}

// AddPipelineRegistrationRaw adds a pipeline registration.
func (c *Client) AddPipelineRegistrationRaw(ctx context.Context, form AddPipelineForm) (any, error) {
	body, err := c.sendRequest(ctx, "POST", "/api/v1/pipelines/add", form)
	if err != nil {
		return nil, fmt.Errorf("adding pipeline registration: %w", err)
	}
	return decodeUntypedJSON(body)
}

// DeletePipelineRegistration removes a pipeline registration.
func (c *Client) DeletePipelineRegistration(ctx context.Context, form DeletePipelineForm) error {
	_, err := c.sendRequest(ctx, "DELETE", "/api/v1/pipelines/delete", form)
	if err != nil {
		return fmt.Errorf("deleting pipeline registration: %w", err)
	}
	return nil
}

// UploadPipelineFileRaw uploads a pipeline file as multipart/form-data.
func (c *Client) UploadPipelineFileRaw(ctx context.Context, filePath string, urlIdx int) (any, error) {
	// #nosec G304 -- filePath is an explicit CLI upload argument.
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening pipeline file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("urlIdx", strconv.Itoa(urlIdx)); err != nil {
		return nil, fmt.Errorf("writing urlIdx form field: %w", err)
	}
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("creating multipart file part: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copying file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s%s", c.BaseURL, "/api/v1/pipelines/upload"), &body)
	if err != nil {
		return nil, fmt.Errorf("error creating upload request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending upload request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("error reading upload response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: status code %d, body: %s", resp.StatusCode, string(respBody))
	}

	return decodeUntypedJSON(respBody)
}

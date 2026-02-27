package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// defaultTimeout is the fallback HTTP client timeout when none is configured.
	defaultTimeout = 30
	// maxResponseSize limits response body reads to 10 MB.
	maxResponseSize = 10 << 20
)

// Client represents the Open WebUI API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new API client abstraction to interact with Open WebUI instances
func NewClient(baseURL, apiKey string, timeoutSeconds int) *Client {
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeout
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// sendRequest is a helper for sending authenticated HTTP requests to the API
func (c *Client) sendRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s%s", c.BaseURL, endpoint), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: status code %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Healthcheck calls the /health endpoint to verify the instance is responsive
func (c *Client) Healthcheck(ctx context.Context) error {
	_, err := c.sendRequest(ctx, "GET", "/health", nil)
	return err
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:3000", "test-key", 10)

	if client.BaseURL != "http://localhost:3000" {
		t.Errorf("expected BaseURL %q, got %q", "http://localhost:3000", client.BaseURL)
	}
	if client.APIKey != "test-key" {
		t.Errorf("expected APIKey %q, got %q", "test-key", client.APIKey)
	}
}

func TestSendRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		endpoint       string
		apiKey         string
		reqAuthHeader  string
		expectedStatus int
		expectedBody   string
		expectError    bool
	}{
		{
			name:           "successful get request",
			method:         "GET",
			endpoint:       "/api/v1/test",
			apiKey:         "secret",
			reqAuthHeader:  "Bearer secret",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
			expectError:    false,
		},
		{
			name:           "unauthorized request",
			method:         "GET",
			endpoint:       "/api/v1/test",
			apiKey:         "wrong-key",
			reqAuthHeader:  "Bearer secret",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"detail":"unauthorized"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != tt.reqAuthHeader {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"detail":"unauthorized"}`))
					return
				}
				w.WriteHeader(tt.expectedStatus)
				w.Write([]byte(tt.expectedBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, tt.apiKey, 1)
			resp, err := client.sendRequest(tt.method, tt.endpoint, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(resp) != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, string(resp))
				}
			}
		})
	}
}

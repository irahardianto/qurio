package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qurio/apps/backend/internal/retrieval"
)

// MockRetriever implements Retriever interface
type MockRetriever struct{}

func (m *MockRetriever) Search(ctx context.Context, query string) ([]retrieval.SearchResult, error) {
	return []retrieval.SearchResult{}, nil
}

type ErrorWrapper struct {
	Status string `json:"status"`
	Error  struct {
		Code    interface{} `json:"code"`
		Message string      `json:"message"`
	} `json:"error"`
	CorrelationID string `json:"correlationId"`
}

func TestHandleMessage_ErrorJSON(t *testing.T) {
	handler := NewHandler(&MockRetriever{})

	tests := []struct {
		name           string
		url            string
		body           string
		expectedStatus int
	}{
		{
			name:           "Missing SessionID",
			url:            "/mcp/message",
			body:           `{"jsonrpc":"2.0", "method":"ping", "id":1}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			url:            "/mcp/message?sessionId=123",
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tt.url, bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			handler.HandleMessage(w, req)

			resp := w.Result()
			
			// Verify Content-Type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify X-Correlation-ID header
			if resp.Header.Get("X-Correlation-ID") == "" {
				t.Error("Expected X-Correlation-ID header")
			}

			// Verify Body Structure
			var errResp ErrorWrapper
			if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
				t.Fatalf("Failed to decode JSON response: %v", err)
			}

			if errResp.Status != "error" {
				t.Errorf("Expected status 'error', got '%s'", errResp.Status)
			}
			
			if errResp.CorrelationID == "" {
				t.Error("Expected CorrelationID in body")
			}
		})
	}
}

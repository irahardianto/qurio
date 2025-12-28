package job

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// MockRepo for Handler Test
type MockRepo struct {
	Repository // Embed interface to skip implementing all methods
}

func (m *MockRepo) List(ctx context.Context) ([]Job, error) {
	return []Job{{ID: "1"}}, nil
}

func TestHandler_List(t *testing.T) {
	repo := &MockRepo{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(repo, nil, logger) // Publisher not needed for List
	handler := NewHandler(service)

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := body["data"]; !ok {
		t.Error("Response missing 'data' field")
	}
	if _, ok := body["meta"]; !ok {
		t.Error("Response missing 'meta' field")
	}
}
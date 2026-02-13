package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/retrieval"
)

// Mocks
type mockRetriever struct{}

func (m *mockRetriever) Search(ctx context.Context, query string, opts *retrieval.SearchOptions) ([]retrieval.SearchResult, error) {
	return []retrieval.SearchResult{}, nil
}

func (m *mockRetriever) GetChunksByURL(ctx context.Context, url string) ([]retrieval.SearchResult, error) {
	return []retrieval.SearchResult{}, nil
}

type mockSourceMgr struct{}

func (m *mockSourceMgr) List(ctx context.Context) ([]source.Source, error) {
	return []source.Source{}, nil
}

func (m *mockSourceMgr) GetPages(ctx context.Context, id string) ([]source.SourcePage, error) {
	return []source.SourcePage{}, nil
}

func TestServeHTTP_Streaming(t *testing.T) {
	handler := NewHandler(&mockRetriever{}, &mockSourceMgr{})

	// Single JSON-RPC request
	reqBody := `{"jsonrpc":"2.0","method":"ping","id":1}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// We no longer expect chunked by default for single request

	decoder := json.NewDecoder(rec.Body)

	// First Response
	var resp1 JSONRPCResponse
	err := decoder.Decode(&resp1)
	assert.NoError(t, err)
	assert.Equal(t, float64(1), resp1.ID)

	// Ensure no second response or garbage
	var resp2 JSONRPCResponse
	err = decoder.Decode(&resp2)
	assert.Error(t, err) // Should be EOF
	assert.Equal(t, io.EOF, err)
}

func TestServeHTTP_Error(t *testing.T) {
	handler := NewHandler(&mockRetriever{}, &mockSourceMgr{})

	// Malformed JSON
	reqBody := `{"jsonrpc":"2.0", "bad":`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// We might get 200 OK with Error body (JSON-RPC) or 200 with error object
	// Our writeError implementation writes 200 OK with error body
	assert.Equal(t, http.StatusOK, rec.Code)

	decoder := json.NewDecoder(rec.Body)
	var resp JSONRPCResponse
	err := decoder.Decode(&resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.Error)
	if errMap, ok := resp.Error.(map[string]interface{}); ok {
		assert.Equal(t, float64(ErrParse), errMap["code"].(float64))
	} else {
		t.Fail()
	}
}

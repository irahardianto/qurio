package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/retrieval"
)

// MockRetriever
type MockRetriever struct {
	mock.Mock
}

func (m *MockRetriever) Search(ctx context.Context, query string, opts *retrieval.SearchOptions) ([]retrieval.SearchResult, error) {
	args := m.Called(ctx, query, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]retrieval.SearchResult), args.Error(1)
}

func (m *MockRetriever) GetChunksByURL(ctx context.Context, url string) ([]retrieval.SearchResult, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]retrieval.SearchResult), args.Error(1)
}

// MockSourceManager
type MockSourceManager struct {
	mock.Mock
}

func (m *MockSourceManager) List(ctx context.Context) ([]source.Source, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]source.Source), args.Error(1)
}

func (m *MockSourceManager) GetPages(ctx context.Context, id string) ([]source.SourcePage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]source.SourcePage), args.Error(1)
}

func TestHandler_ServeHTTP_Initialize(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := NewHandler(mockRetriever, mockSourceMgr)

	reqBody := `{"jsonrpc": "2.0", "method": "initialize", "id": 1}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsonResp JSONRPCResponse
	json.NewDecoder(resp.Body).Decode(&jsonResp)
	assert.Equal(t, "2.0", jsonResp.JSONRPC)
}

func TestHandler_ServeHTTP_ListTools(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := NewHandler(mockRetriever, mockSourceMgr)

	reqBody := `{"jsonrpc": "2.0", "method": "tools/list", "id": 1}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var jsonResp JSONRPCResponse
	json.NewDecoder(w.Result().Body).Decode(&jsonResp)
	assert.NotNil(t, jsonResp.Result)
}

func TestHandler_ServeHTTP_CallSearch(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := NewHandler(mockRetriever, mockSourceMgr)

	mockRetriever.On("Search", mock.Anything, "test", mock.Anything).Return([]retrieval.SearchResult{
		{Content: "test content", Score: 0.9},
	}, nil)

	reqBody := `{
		"jsonrpc": "2.0", 
		"method": "tools/call", 
		"id": 1, 
		"params": {
			"name": "qurio_search",
			"arguments": {
				"query": "test"
			}
		}
	}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var jsonResp JSONRPCResponse
	json.NewDecoder(w.Result().Body).Decode(&jsonResp)
	assert.Nil(t, jsonResp.Error)
}

func TestHandle_ListSources(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSrc := new(MockSourceManager)
	
	mockSrc.On("List", mock.Anything).Return([]source.Source{
		{ID: "src_1", Name: "Docs", Type: "web"},
		{ID: "src_2", Name: "", URL: "http://example.com", Type: "web"},
	}, nil)

	h := NewHandler(mockRetriever, mockSrc)
	
	// 1. Verify Tool Exists
	reqList := JSONRPCRequest{
		Method: "tools/list",
		ID:     1,
	}
	respList := h.processRequest(context.Background(), reqList)
	listRes := respList.Result.(ListToolsResult)
	found := false
	for _, tool := range listRes.Tools {
		if tool.Name == "qurio_list_sources" {
			found = true
			break
		}
	}
	if !found {
		t.Error("qurio_list_sources tool not found in list")
	}

	// 2. Verify Call
	reqCall := JSONRPCRequest{
		Method: "tools/call",
		Params: json.RawMessage(`{"name": "qurio_list_sources", "arguments": {}}`),
		ID:     2,
	}
	
	respCall := h.processRequest(context.Background(), reqCall)
	if respCall.Error != nil {
		t.Errorf("Unexpected error: %v", respCall.Error)
	}
	
	res := respCall.Result.(ToolResult)
	if len(res.Content) == 0 {
		t.Fatal("No content returned")
	}
	
	// Check if JSON contains src_1
	if !strings.Contains(res.Content[0].Text, "src_1") {
		t.Errorf("Expected output to contain src_1, got: %s", res.Content[0].Text)
	}
}

func TestHandle_ListPages(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSrc := new(MockSourceManager)
	
	mockSrc.On("GetPages", mock.Anything, "src_1").Return([]source.SourcePage{
		{ID: "page_1", SourceID: "src_1", URL: "/home", Status: "completed"},
	}, nil)

	h := NewHandler(mockRetriever, mockSrc)

	// 2. Verify Call
	reqCall := JSONRPCRequest{
		Method: "tools/call",
		Params: json.RawMessage(`{"name": "qurio_list_pages", "arguments": {"source_id": "src_1"}}`),
		ID:     2,
	}
	
	respCall := h.processRequest(context.Background(), reqCall)
	if respCall.Error != nil {
		t.Errorf("Unexpected error: %v", respCall.Error)
	}
	
	res := respCall.Result.(ToolResult)
	if len(res.Content) == 0 {
		t.Fatal("No content returned")
	}
	
	if !strings.Contains(res.Content[0].Text, "/home") {
		t.Errorf("Expected output to contain /home, got: %s", res.Content[0].Text)
	}
}

func TestHandle_Search_WithSourceID(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSrc := new(MockSourceManager)

	mockRetriever.On("Search", mock.Anything, "test", mock.MatchedBy(func(opts *retrieval.SearchOptions) bool {
		val, ok := opts.Filters["sourceId"]
		return ok && val == "src_123"
	})).Return([]retrieval.SearchResult{}, nil)

	h := NewHandler(mockRetriever, mockSrc)

	req := JSONRPCRequest{
		Method: "tools/call",
		Params: json.RawMessage(`{"name": "qurio_search", "arguments": {"query": "test", "source_id": "src_123"}}`),
		ID:     1,
	}
	
	resp := h.processRequest(context.Background(), req)
	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
	mockRetriever.AssertExpectations(t)
}

func TestHandler_HandleMessage(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := NewHandler(mockRetriever, mockSourceMgr)

	reqBody := `{"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}}`
	
	// Create request with sessionId
	req := httptest.NewRequest("POST", "/mcp/messages?sessionId=invalid", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleMessage(w, req)

	// Should return 404 because session not found
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestHandler_HandleMessage_Validation(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := NewHandler(mockRetriever, mockSourceMgr)

	req := httptest.NewRequest("POST", "/mcp/messages", nil) // Missing sessionId
	w := httptest.NewRecorder()

	handler.HandleMessage(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandler_HandleMessage_Success(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := NewHandler(mockRetriever, mockSourceMgr)

	reqSSE := httptest.NewRequest("GET", "/mcp/sse", nil)
	wSSE := httptest.NewRecorder()
	
	ctx, cancel := context.WithCancel(context.Background())
	reqSSE = reqSSE.WithContext(ctx)
	
	// Start SSE in goroutine
	go handler.HandleSSE(wSSE, reqSSE)
	
	// Terminate
	cancel()
}

func TestHandler_ProcessRequest_Table(t *testing.T) {
	// Define Cases
	tests := []struct {
		name    string
		req     JSONRPCRequest
		setup   func(mr *MockRetriever, msm *MockSourceManager)
		wantRes func(*JSONRPCResponse) bool
		wantErr bool
	}{
		{
			name: "Initialize",
			req:  JSONRPCRequest{Method: "initialize", ID: 1},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {},
			wantRes: func(r *JSONRPCResponse) bool {
				res := r.Result.(map[string]interface{})
				return res["protocolVersion"] == "2024-11-05"
			},
		},
		{
			name: "List Tools",
			req:  JSONRPCRequest{Method: "tools/list", ID: 2},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {},
			wantRes: func(r *JSONRPCResponse) bool {
				 res := r.Result.(ListToolsResult)
				 return len(res.Tools) > 0
			},
		},
		{
			 name: "Call Unknown Tool",
			 req: JSONRPCRequest{Method: "tools/call", Params: json.RawMessage(`{"name": "unknown"}`), ID: 3},
			 setup: func(mr *MockRetriever, msm *MockSourceManager) {},
			 wantRes: func(r *JSONRPCResponse) bool {
				 errMap := r.Error.(map[string]interface{})
				 return errMap["code"].(int) == ErrMethodNotFound
			 },
		},
		{
			name: "List Sources",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_list_sources", "arguments": {}}`),
				ID: 4,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {
				msm.On("List", mock.Anything).Return([]source.Source{{ID: "s1", Name: "Source 1", Type: "web"}}, nil)
			},
			wantRes: func(r *JSONRPCResponse) bool {
				res := r.Result.(ToolResult)
				return strings.Contains(res.Content[0].Text, "s1")
			},
		},
		{
			name: "List Pages",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_list_pages", "arguments": {"source_id": "s1"}}`),
				ID: 5,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {
				msm.On("GetPages", mock.Anything, "s1").Return([]source.SourcePage{{ID: "p1", URL: "/page1"}}, nil)
			},
			wantRes: func(r *JSONRPCResponse) bool {
				res := r.Result.(ToolResult)
				return strings.Contains(res.Content[0].Text, "/page1")
			},
		},
		{
			name: "Search",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_search", "arguments": {"query": "test"}}`),
				ID: 6,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {
				mr.On("Search", mock.Anything, "test", mock.Anything).Return([]retrieval.SearchResult{{Content: "found"}}, nil)
			},
			wantRes: func(r *JSONRPCResponse) bool {
				res := r.Result.(ToolResult)
				return strings.Contains(res.Content[0].Text, "found")
			},
		},
		{
			name: "Read Page",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_read_page", "arguments": {"url": "http://example.com"}}`),
				ID: 7,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {
				mr.On("GetChunksByURL", mock.Anything, "http://example.com").Return([]retrieval.SearchResult{{Content: "page content"}}, nil)
			},
			wantRes: func(r *JSONRPCResponse) bool {
				res := r.Result.(ToolResult)
				return strings.Contains(res.Content[0].Text, "page content")
			},
		},
		{
			name: "Search Invalid Params (Empty Query)",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_search", "arguments": {}}`),
				ID: 8,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {},
			wantRes: func(r *JSONRPCResponse) bool {
				if r.Error == nil { return false }
				errMap := r.Error.(map[string]interface{})
				return errMap["code"].(int) == ErrInvalidParams
			},
		},
		{
			name: "Search Invalid Params (Alpha Out of Range)",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_search", "arguments": {"query": "test", "alpha": 1.5}}`),
				ID: 81,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {},
			wantRes: func(r *JSONRPCResponse) bool {
				if r.Error == nil { return false }
				errMap := r.Error.(map[string]interface{})
				return errMap["code"].(int) == ErrInvalidParams
			},
		},
		{
			name: "Read Page Invalid Params (Empty URL)",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_read_page", "arguments": {"url": ""}}`),
				ID: 9,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {},
			wantRes: func(r *JSONRPCResponse) bool {
				if r.Error == nil { return false }
				errMap := r.Error.(map[string]interface{})
				return errMap["code"].(int) == ErrInvalidParams
			},
		},
		{
			name: "Read Page No Content",
			req: JSONRPCRequest{
				Method: "tools/call",
				Params: json.RawMessage(`{"name": "qurio_read_page", "arguments": {"url": "http://empty.com"}}`),
				ID: 10,
			},
			setup: func(mr *MockRetriever, msm *MockSourceManager) {
				mr.On("GetChunksByURL", mock.Anything, "http://empty.com").Return([]retrieval.SearchResult{}, nil)
			},
			wantRes: func(r *JSONRPCResponse) bool {
				res := r.Result.(ToolResult)
				return strings.Contains(res.Content[0].Text, "No content found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := new(MockRetriever)
			msm := new(MockSourceManager)
			handler := NewHandler(mr, msm)
			
			tt.setup(mr, msm)
			
			// Use processRequest directly as it's cleaner for unit tests
			res := handler.processRequest(context.Background(), tt.req)
			if tt.wantErr {
				assert.Nil(t, res)
			} else {
				assert.NotNil(t, res)
				if !tt.wantRes(res) {
					t.Errorf("wantRes failed for %s", tt.name)
				}
			}
		})
	}
}
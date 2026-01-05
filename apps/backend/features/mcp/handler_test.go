package mcp

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "net/http"
    "net/http/httptest"
    "strings"
    "time"
    "encoding/json"
    "qurio/apps/backend/internal/retrieval"
    "qurio/apps/backend/internal/middleware"
)

// MockRetriever
type MockRetriever struct {
    mock.Mock
}

func (m *MockRetriever) Search(ctx context.Context, query string, opts *retrieval.SearchOptions) ([]retrieval.SearchResult, error) {
    args := m.Called(ctx, query, opts)
    return args.Get(0).([]retrieval.SearchResult), args.Error(1)
}

func (m *MockRetriever) GetChunksByURL(ctx context.Context, url string) ([]retrieval.SearchResult, error) {
    args := m.Called(ctx, url)
    return args.Get(0).([]retrieval.SearchResult), args.Error(1)
}

func TestToolsList_ReturnsQurioTools(t *testing.T) {
    mockRetriever := new(MockRetriever)
    handler := NewHandler(mockRetriever, new(MockSourceManager))

    reqBody := `{"jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": 1}`
    // Note: processRequest is internal, but we can test ServeHTTP or simulate it. 
    // Testing HandleMessage is tricky due to async SSE.
    // However, the test file has access to internal methods if it's in package mcp (it is).
    // So we can call processRequest directly if we expose it or use ServeHTTP.
    
    // Using processRequest directly (since it's in same package)
    var req JSONRPCRequest
    json.Unmarshal([]byte(reqBody), &req)
    resp := handler.processRequest(context.Background(), req)
    
    assert.NotNil(t, resp.Result)
    listResult := resp.Result.(ListToolsResult)
    
    toolNames := make(map[string]bool)
    for _, tool := range listResult.Tools {
        toolNames[tool.Name] = true
    }
    
    assert.True(t, toolNames["qurio_search"])
    assert.True(t, toolNames["qurio_fetch_page"])
}

func TestHandleMessage_ContextPropagation(t *testing.T) {
    // Setup
    mockRetriever := new(MockRetriever)
    handler := NewHandler(mockRetriever, new(MockSourceManager))
    
    // Create a session
    wSSE := httptest.NewRecorder()
    rSSE := httptest.NewRequest("GET", "/mcp/sse", nil)
    go handler.HandleSSE(wSSE, rSSE)
    
    // Wait for session to be established
    time.Sleep(100 * time.Millisecond)
    
    // Find the session ID
    var sessionID string
    handler.sessionsLock.RLock()
    for k := range handler.sessions {
        sessionID = k
        break
    }
    handler.sessionsLock.RUnlock()
    
    assert.NotEmpty(t, sessionID, "Session ID should have been created")

    // Create a request with correlation ID
    // Updated tool name to "qurio_search"
    reqBody := `{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "qurio_search", "arguments": {"query": "test"}}, "id": 1}`
    r := httptest.NewRequest("POST", "/mcp/messages?sessionId="+sessionID, strings.NewReader(reqBody))
    
    correlationID := "test-correlation-id-123"
    // Use helper to set context
    ctx := middleware.WithCorrelationID(r.Context(), correlationID)
    r = r.WithContext(ctx)
    w := httptest.NewRecorder()

    // Expectation: Search is called with a context containing the correlation ID
    mockRetriever.On("Search", mock.MatchedBy(func(ctx context.Context) bool {
        // Verify correlation ID is present using helper
        val := middleware.GetCorrelationID(ctx)
        return val == correlationID
    }), "test", mock.Anything).Return([]retrieval.SearchResult{}, nil)

    // Act
    handler.HandleMessage(w, r)
    
    // Verify immediate response
    assert.Equal(t, http.StatusAccepted, w.Code)
    
    // Wait for async processing
    time.Sleep(100 * time.Millisecond)
    
    // Assert
    mockRetriever.AssertExpectations(t)
}

func TestToolsCall_FetchPage(t *testing.T) {
    mockRetriever := new(MockRetriever)
    handler := NewHandler(mockRetriever, new(MockSourceManager))

    url := "http://example.com"
    mockRetriever.On("GetChunksByURL", mock.Anything, url).Return([]retrieval.SearchResult{
        {Content: "Chunk 1", Metadata: map[string]interface{}{"url": url}},
        {Content: "Chunk 2", Metadata: map[string]interface{}{"url": url}},
    }, nil)

    reqBody := `{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "qurio_fetch_page", "arguments": {"url": "http://example.com"}}, "id": 1}`
    
    var req JSONRPCRequest
    json.Unmarshal([]byte(reqBody), &req)
    
    resp := handler.processRequest(context.Background(), req)
    
    assert.Nil(t, resp.Error)
    result := resp.Result.(ToolResult)
    assert.Contains(t, result.Content[0].Text, "Chunk 1")
    assert.Contains(t, result.Content[0].Text, "Chunk 2")
    mockRetriever.AssertExpectations(t)
}
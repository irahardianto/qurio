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

func TestHandleMessage_ContextPropagation(t *testing.T) {
    // Setup
    mockRetriever := new(MockRetriever)
    handler := NewHandler(mockRetriever)
    
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
    reqBody := `{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "search", "arguments": {"query": "test"}}, "id": 1}`
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

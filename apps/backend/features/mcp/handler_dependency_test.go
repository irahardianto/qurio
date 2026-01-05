package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/retrieval"
	"github.com/stretchr/testify/mock"
)

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

func TestNewHandler_WithSourceManager(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)

	// This call will fail compilation until we update NewHandler signature
	h := NewHandler(mockRetriever, mockSourceMgr)
	
	if h == nil {
		t.Fatal("Handler should not be nil")
	}
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
	
	// Check fallback for src_2 (Name should be URL)
	if !strings.Contains(res.Content[0].Text, "http://example.com") {
		t.Errorf("Expected output to contain fallback URL for src_2, got: %s", res.Content[0].Text)
	}
}

func TestHandle_ListPages(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSrc := new(MockSourceManager)
	
	mockSrc.On("GetPages", mock.Anything, "src_1").Return([]source.SourcePage{
		{ID: "page_1", SourceID: "src_1", URL: "/home", Status: "completed"},
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
		if tool.Name == "qurio_list_pages" {
			found = true
			break
		}
	}
	if !found {
		t.Error("qurio_list_pages tool not found in list")
	}

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

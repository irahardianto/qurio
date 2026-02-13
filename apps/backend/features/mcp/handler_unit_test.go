package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"qurio/apps/backend/features/mcp"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/retrieval"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRetriever implements mcp.Retriever
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

// MockSourceManager implements mcp.SourceManager
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

func TestProcessRequest_Initialize(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 1, resp.ID)
	assert.NotNil(t, resp.Result)

	result := resp.Result.(map[string]interface{})
	assert.Equal(t, "2024-11-05", result["protocolVersion"])
	assert.NotNil(t, result["capabilities"])
	assert.NotNil(t, result["serverInfo"])
}

func TestProcessRequest_NotificationsInitialized(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	resp := handler.ProcessRequest(context.Background(), req)

	// Notifications must not generate a response
	assert.Nil(t, resp)
}

func TestProcessRequest_ToolsList(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      2,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 2, resp.ID)
	assert.NotNil(t, resp.Result)

	result := resp.Result.(mcp.ListToolsResult)
	assert.Len(t, result.Tools, 4)

	toolNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		toolNames[i] = tool.Name
	}
	assert.Contains(t, toolNames, "qurio_search")
	assert.Contains(t, toolNames, "qurio_list_sources")
	assert.Contains(t, toolNames, "qurio_list_pages")
	assert.Contains(t, toolNames, "qurio_read_page")
}

func TestProcessRequest_QuriSearch_Success(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	searchResults := []retrieval.SearchResult{
		{
			Content:  "Test content",
			Title:    "Test Title",
			Score:    0.95,
			Type:     "code",
			Language: "go",
			SourceID: "src1",
		},
	}

	mockRetriever.On("Search", mock.Anything, "test query", mock.Anything).Return(searchResults, nil)

	args := map[string]interface{}{
		"query": "test query",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_search",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      3,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.NotEmpty(t, result.Content)
	assert.Contains(t, result.Content[0].Text, "Test Title")
	assert.Contains(t, result.Content[0].Text, "Test content")

	mockRetriever.AssertExpectations(t)
}

func TestProcessRequest_QuriSearch_MissingQuery(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	args := map[string]interface{}{}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_search",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      4,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)

	errMap := resp.Error.(map[string]interface{})
	assert.Equal(t, mcp.ErrInvalidParams, errMap["code"])
	assert.Contains(t, errMap["message"], "Query is required")
}

func TestProcessRequest_QuriSearch_InvalidAlpha(t *testing.T) {
	tests := []struct {
		name  string
		alpha float32
	}{
		{"Alpha too low", -0.1},
		{"Alpha too high", 1.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRetriever := new(MockRetriever)
			mockSourceMgr := new(MockSourceManager)
			handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

			args := map[string]interface{}{
				"query": "test",
				"alpha": tt.alpha,
			}
			argsJSON, _ := json.Marshal(args)

			params := mcp.CallParams{
				Name:      "qurio_search",
				Arguments: argsJSON,
			}
			paramsJSON, _ := json.Marshal(params)

			req := mcp.JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params:  paramsJSON,
				ID:      5,
			}

			resp := handler.ProcessRequest(context.Background(), req)

			assert.NotNil(t, resp)
			assert.NotNil(t, resp.Error)

			errMap := resp.Error.(map[string]interface{})
			assert.Equal(t, mcp.ErrInvalidParams, errMap["code"])
			assert.Contains(t, errMap["message"], "Alpha must be between 0.0 and 1.0")
		})
	}
}

func TestProcessRequest_QuriSearch_WithFilters(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockRetriever.On("Search", mock.Anything, "test", mock.MatchedBy(func(opts *retrieval.SearchOptions) bool {
		return opts.Filters != nil &&
			opts.Filters["type"] == "code" &&
			opts.Filters["language"] == "go"
	})).Return([]retrieval.SearchResult{}, nil)

	args := map[string]interface{}{
		"query": "test",
		"filters": map[string]interface{}{
			"type":     "code",
			"language": "go",
		},
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_search",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      6,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	mockRetriever.AssertExpectations(t)
}

func TestProcessRequest_QuriSearch_WithSourceID(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	sourceID := "src123"
	mockRetriever.On("Search", mock.Anything, "test", mock.MatchedBy(func(opts *retrieval.SearchOptions) bool {
		return opts.Filters != nil && opts.Filters["sourceId"] == sourceID
	})).Return([]retrieval.SearchResult{}, nil)

	args := map[string]interface{}{
		"query":     "test",
		"source_id": sourceID,
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_search",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      7,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	mockRetriever.AssertExpectations(t)
}

func TestProcessRequest_QuriSearch_SearchError(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockRetriever.On("Search", mock.Anything, "test", mock.Anything).Return(nil, assert.AnError)

	args := map[string]interface{}{
		"query": "test",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_search",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      8,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)

	errMap := resp.Error.(map[string]interface{})
	assert.Equal(t, mcp.ErrInternal, errMap["code"])
	assert.Contains(t, errMap["message"], "Search failed")
}

func TestProcessRequest_QuriSearch_NoResults(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockRetriever.On("Search", mock.Anything, "test", mock.Anything).Return([]retrieval.SearchResult{}, nil)

	args := map[string]interface{}{
		"query": "test",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_search",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      9,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "No results found")
}

func TestProcessRequest_QuriListSources_Success(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	sources := []source.Source{
		{ID: "src1", Name: "Test Source", Type: "web", URL: "http://example.com"},
	}
	mockSourceMgr.On("List", mock.Anything).Return(sources, nil)

	params := mcp.CallParams{
		Name:      "qurio_list_sources",
		Arguments: json.RawMessage("{}"),
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      10,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "src1")
	assert.Contains(t, result.Content[0].Text, "Test Source")

	mockSourceMgr.AssertExpectations(t)
}

func TestProcessRequest_QuriListSources_Empty(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockSourceMgr.On("List", mock.Anything).Return([]source.Source{}, nil)

	params := mcp.CallParams{
		Name:      "qurio_list_sources",
		Arguments: json.RawMessage("{}"),
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      11,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "No sources found")
}

func TestProcessRequest_QuriListSources_Error(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockSourceMgr.On("List", mock.Anything).Return(nil, assert.AnError)

	params := mcp.CallParams{
		Name:      "qurio_list_sources",
		Arguments: json.RawMessage("{}"),
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      12,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error) // Error is in result, not response error

	result := resp.Result.(mcp.ToolResult)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "Error:")
}

func TestProcessRequest_QuriListPages_Success(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	pages := []source.SourcePage{
		{ID: "page1", URL: "http://example.com/page1"},
	}
	mockSourceMgr.On("GetPages", mock.Anything, "src1").Return(pages, nil)

	args := map[string]interface{}{
		"source_id": "src1",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_list_pages",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      13,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "page1")

	mockSourceMgr.AssertExpectations(t)
}

func TestProcessRequest_QuriListPages_MissingSourceID(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	args := map[string]interface{}{}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_list_pages",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      14,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)

	errMap := resp.Error.(map[string]interface{})
	assert.Equal(t, mcp.ErrInvalidParams, errMap["code"])
	assert.Contains(t, errMap["message"], "source_id is required")
}

func TestProcessRequest_QuriListPages_Empty(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockSourceMgr.On("GetPages", mock.Anything, "src1").Return([]source.SourcePage{}, nil)

	args := map[string]interface{}{
		"source_id": "src1",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_list_pages",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      15,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "No pages found")
}

func TestProcessRequest_QuriListPages_Error(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockSourceMgr.On("GetPages", mock.Anything, "src1").Return(nil, assert.AnError)

	args := map[string]interface{}{
		"source_id": "src1",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_list_pages",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      16,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "Error:")
}

func TestProcessRequest_QuriReadPage_Success(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	chunks := []retrieval.SearchResult{
		{Content: "Page content", Title: "Page Title", Type: "prose"},
		{Content: "func main() {}", Language: "go", Type: "code"},
	}
	mockRetriever.On("GetChunksByURL", mock.Anything, "http://example.com").Return(chunks, nil)

	args := map[string]interface{}{
		"url": "http://example.com",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_read_page",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      17,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "Page Title")
	assert.Contains(t, result.Content[0].Text, "Page content")
	assert.Contains(t, result.Content[0].Text, "func main()")

	mockRetriever.AssertExpectations(t)
}

func TestProcessRequest_QuriReadPage_MissingURL(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	args := map[string]interface{}{}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_read_page",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      18,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)

	errMap := resp.Error.(map[string]interface{})
	assert.Equal(t, mcp.ErrInvalidParams, errMap["code"])
	assert.Contains(t, errMap["message"], "URL is required")
}

func TestProcessRequest_QuriReadPage_NoContent(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockRetriever.On("GetChunksByURL", mock.Anything, "http://example.com").Return([]retrieval.SearchResult{}, nil)

	args := map[string]interface{}{
		"url": "http://example.com",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_read_page",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      19,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "No content found")
}

func TestProcessRequest_QuriReadPage_Error(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	mockRetriever.On("GetChunksByURL", mock.Anything, "http://example.com").Return(nil, assert.AnError)

	args := map[string]interface{}{
		"url": "http://example.com",
	}
	argsJSON, _ := json.Marshal(args)

	params := mcp.CallParams{
		Name:      "qurio_read_page",
		Arguments: argsJSON,
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      20,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result := resp.Result.(mcp.ToolResult)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "Error:")
}

func TestProcessRequest_UnknownMethod(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	params := mcp.CallParams{
		Name:      "unknown_tool",
		Arguments: json.RawMessage("{}"),
	}
	paramsJSON, _ := json.Marshal(params)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsJSON,
		ID:      21,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)

	errMap := resp.Error.(map[string]interface{})
	assert.Equal(t, mcp.ErrMethodNotFound, errMap["code"])
	assert.Contains(t, errMap["message"], "Method not found")
}

func TestProcessRequest_InvalidJSONRPCMethod(t *testing.T) {
	mockRetriever := new(MockRetriever)
	mockSourceMgr := new(MockSourceManager)
	handler := mcp.NewHandler(mockRetriever, mockSourceMgr)

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "invalid_method",
		ID:      22,
	}

	resp := handler.ProcessRequest(context.Background(), req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)

	errMap := resp.Error.(map[string]interface{})
	assert.Equal(t, mcp.ErrMethodNotFound, errMap["code"])
}

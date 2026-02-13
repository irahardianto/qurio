package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/features/mcp"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/middleware"
	"qurio/apps/backend/internal/retrieval"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/testutils"
	"qurio/apps/backend/internal/worker"
)

// MockEmbedder
type MockEmbedder struct {
	mock.Mock
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

// SpyRetriever for context verification
type SpyRetriever struct {
	mock.Mock
	LastCtx context.Context
}

func (m *SpyRetriever) Search(ctx context.Context, query string, opts *retrieval.SearchOptions) ([]retrieval.SearchResult, error) {
	m.LastCtx = ctx
	return []retrieval.SearchResult{}, nil
}

func (m *SpyRetriever) GetChunksByURL(ctx context.Context, url string) ([]retrieval.SearchResult, error) {
	m.LastCtx = ctx
	return []retrieval.SearchResult{}, nil
}

func TestMCPHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	ctx := context.Background()

	// 1. Setup Dependencies
	vectorStore := weaviate.NewStore(s.Weaviate)
	require.NoError(t, vectorStore.EnsureSchema(ctx))

	embedder := new(MockEmbedder)
	settingsRepo := settings.NewPostgresRepo(s.DB)
	settingsSvc := settings.NewService(settingsRepo)
	retrievalSvc := retrieval.NewService(embedder, vectorStore, nil, settingsSvc, nil)
	sourceRepo := source.NewPostgresRepo(s.DB)

	handler := mcp.NewHandler(retrievalSvc, sourceRepo)

	// 2. Seed Data
	src := &source.Source{
		Type:        "web",
		URL:         "http://example.com",
		ContentHash: "hash-mcp",
		Status:      "completed",
		Name:        "MCP Test Source",
	}
	err := sourceRepo.Save(ctx, src)
	require.NoError(t, err)

	_, err = sourceRepo.BulkCreatePages(ctx, []source.SourcePage{{
		SourceID: src.ID,
		URL:      src.URL,
		Status:   "completed",
		Depth:    0,
	}})
	require.NoError(t, err)

	chunk := worker.Chunk{
		SourceID:   src.ID,
		SourceURL:  src.URL,
		Content:    "The quick brown fox jumps over the lazy dog.",
		ChunkIndex: 0,
		Title:      "Fox Page",
		Type:       "web",
		Vector:     []float32{0.1, 0.2, 0.3},
	}
	err = vectorStore.StoreChunk(ctx, chunk)
	require.NoError(t, err)

	// 3. Test qurio_search via JSON-RPC
	searchArgs := mcp.SearchArgs{
		Query: "fox",
	}
	argsBytes, _ := json.Marshal(searchArgs)

	callParams := mcp.CallParams{
		Name:      "qurio_search",
		Arguments: argsBytes,
	}
	paramsBytes, _ := json.Marshal(callParams)

	reqBody := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsBytes,
		ID:      1,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	// Use simplified path /mcp
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(bodyBytes)))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp mcp.JSONRPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Nil(t, resp.Error)

	resultMap, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	contentList, ok := resultMap["content"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, contentList)

	firstContent := contentList[0].(map[string]interface{})
	text := firstContent["text"].(string)

	assert.Contains(t, text, "Fox Page")
	assert.Contains(t, text, "The quick brown fox")
}

func TestIntegration_Streaming_Correlation(t *testing.T) {
	spyRetriever := &SpyRetriever{}
	handler := mcp.NewHandler(spyRetriever, nil) // sourceMgr nil as we won't call it

	correlationID := "test-correlation-id-streaming"

	// Prepare a request that calls Search, so we can verify context on the spy
	searchArgs := mcp.SearchArgs{Query: "test"}
	argsBytes, _ := json.Marshal(searchArgs)
	callParams := mcp.CallParams{Name: "qurio_search", Arguments: argsBytes}
	paramsBytes, _ := json.Marshal(callParams)
	reqRPC := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsBytes,
		ID:      1,
	}
	bodyBytes, _ := json.Marshal(reqRPC)

	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(bodyBytes)))
	req.Header.Set("X-Correlation-ID", correlationID)

	rec := httptest.NewRecorder()

	// Wrap with middleware
	mw := middleware.CorrelationID(http.HandlerFunc(handler.ServeHTTP))
	mw.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify Context
	require.NotNil(t, spyRetriever.LastCtx, "Retriever should have been called")
	gotID := middleware.GetCorrelationID(spyRetriever.LastCtx)
	assert.Equal(t, correlationID, gotID)
}

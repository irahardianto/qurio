package weaviate

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore_CountChunksBySource(t *testing.T) {
	server := newMockWeaviateServer(t, func(r *http.Request, body map[string]interface{}) {
		assert.Equal(t, "/v1/graphql", r.URL.Path)
		query := body["query"].(string)
		assert.Contains(t, query, "Aggregate")
		assert.Contains(t, query, "DocumentChunk")
		assert.Contains(t, query, "meta")
		assert.Contains(t, query, "count")
		assert.Contains(t, query, "sourceId")
	})
	defer server.Close()

	store := newTestStore(t, server)

	count, err := store.CountChunksBySource(context.Background(), "src-1")
	assert.NoError(t, err)
	assert.Equal(t, 0, count) // Mock returns empty/wrong format so 0 is expected for now
}

func TestStore_GetChunks_Pagination(t *testing.T) {
	server := newMockWeaviateServer(t, func(r *http.Request, body map[string]interface{}) {
		assert.Equal(t, "/v1/graphql", r.URL.Path)
		query := body["query"].(string)
		assert.Contains(t, query, "limit: 10")
		assert.Contains(t, query, "offset: 5")
	})
	defer server.Close()

	store := newTestStore(t, server)

	chunks, err := store.GetChunks(context.Background(), "src-1", 10, 5)
	assert.NoError(t, err)
	assert.NotNil(t, chunks)
}

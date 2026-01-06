package weaviate_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/testutils"
	"qurio/apps/backend/internal/worker"
)

func TestWeaviateStore_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	store := weaviate.NewStore(s.Weaviate)
	ctx := context.Background()

	// Ensure Schema
	err := store.EnsureSchema(ctx)
	require.NoError(t, err)

	// 1. Store & Delete
	chunk := worker.Chunk{
		SourceID:   "src-1",
		SourceURL:  "http://example.com/page",
		Content:    "Postgres is a database",
		ChunkIndex: 0,
		Title:      "My Page",
		Type:       "web",
		Language:   "en",
		// Vector: []float32{0.1, 0.2, ...} // In 'none' vectorizer, we can provide vector or not.
		// Since DEFAULT_VECTORIZER_MODULE is 'none', we might need to provide a dummy vector if the schema expects it?
		// EnsureSchema logic determines if vectorizer is used.
		// Let's assume the adapter handles nil vector if permitted or we don't strictly check vector search quality, just retrieval.
		Vector: []float32{0.1, 0.2, 0.3},
	}
	err = store.StoreChunk(ctx, chunk)
	require.NoError(t, err)

	// Verify existence via Search
	res, err := store.Search(ctx, "Postgres", nil, 0.0, 10, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, res)
	assert.Equal(t, "Postgres is a database", res[0].Content)

	// Delete by URL
	err = store.DeleteChunksByURL(ctx, "src-1", "http://example.com/page")
	require.NoError(t, err)

	// Verify deletion
	res, err = store.Search(ctx, "Postgres", nil, 0.0, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, res)

	// 2. Hybrid Search & Filtering
	chunkA := worker.Chunk{SourceID: "src-2", SourceURL: "u1", Content: "Postgres", ChunkIndex: 0, Vector: []float32{0.1, 0.1, 0.1}, Type: "web"}
	chunkB := worker.Chunk{SourceID: "src-2", SourceURL: "u2", Content: "Databases", ChunkIndex: 0, Vector: []float32{0.2, 0.2, 0.2}, Type: "pdf"}
	err = store.StoreChunk(ctx, chunkA)
	require.NoError(t, err)
	err = store.StoreChunk(ctx, chunkB)
	require.NoError(t, err)

	// Search for "Postgres" with keyword preference (alpha 0.0)
	res, err = store.Search(ctx, "Postgres", []float32{0.1, 0.1, 0.1}, 0.0, 10, nil)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	assert.Equal(t, "Postgres", res[0].Content)
	assert.Equal(t, "web", res[0].Metadata["type"])

	// Search with filter (Type=pdf)
	filters := map[string]interface{}{"type": "pdf"}
	res, err = store.Search(ctx, "Databases", []float32{0.2, 0.2, 0.2}, 0.5, 10, filters)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	assert.Equal(t, "Databases", res[0].Content)
	assert.Equal(t, "pdf", res[0].Metadata["type"])

	// Verify Count
	count, err := store.CountChunks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Delete by SourceID
	err = store.DeleteChunksBySourceID(ctx, "src-2")
	require.NoError(t, err)

	count, err = store.CountChunks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

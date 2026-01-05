package weaviate_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	adapter "qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/worker"
)

func mockWeaviate(t *testing.T, handler http.HandlerFunc) (*weaviate.Client, *httptest.Server) {
	ts := httptest.NewServer(handler)
	cfg := weaviate.Config{Host: ts.Listener.Addr().String(), Scheme: "http"}
	client, err := weaviate.NewClient(cfg)
	assert.NoError(t, err)
	return client, ts
}

func TestStore_StoreChunk(t *testing.T) {
	client, ts := mockWeaviate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/meta" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.19.0"}`))
			return
		}
		assert.Equal(t, "/v1/objects", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		props := body["properties"].(map[string]interface{})
		assert.Equal(t, "test content", props["content"])
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "1"})
	})
	defer ts.Close()

	store := adapter.NewStore(client)
	chunk := worker.Chunk{
		Content: "test content",
		SourceID: "src1",
		ChunkIndex: 0,
		Vector: []float32{0.1, 0.2},
	}
	err := store.StoreChunk(context.Background(), chunk)
	assert.NoError(t, err)
}

func TestStore_DeleteChunksByURL(t *testing.T) {
	client, ts := mockWeaviate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/meta" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.19.0"}`))
			return
		}
		assert.Equal(t, "/v1/batch/objects", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	})
	defer ts.Close()

	store := adapter.NewStore(client)
	err := store.DeleteChunksByURL(context.Background(), "src1", "http://u.rl")
	assert.NoError(t, err)
}

func TestStore_GetChunks(t *testing.T) {
	client, ts := mockWeaviate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/meta" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.19.0"}`))
			return
		}
		assert.Equal(t, "/v1/graphql", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		// Mock GraphQL response
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"Get": map[string]interface{}{
					"DocumentChunk": []interface{}{
						map[string]interface{}{
							"content": "chunk content",
							"chunkIndex": 0.0,
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	store := adapter.NewStore(client)
	chunks, err := store.GetChunks(context.Background(), "src1")
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)
	assert.Equal(t, "chunk content", chunks[0].Content)
}

func TestStore_Search(t *testing.T) {
	client, ts := mockWeaviate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/meta" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.19.0"}`))
			return
		}
		assert.Equal(t, "/v1/graphql", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"Get": map[string]interface{}{
					"DocumentChunk": []interface{}{
						map[string]interface{}{
							"content": "found content",
							"_additional": map[string]interface{}{
								"score": "0.95",
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	store := adapter.NewStore(client)
	results, err := store.Search(context.Background(), "query", []float32{0.1, 0.2}, 0.5, 10, nil)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "found content", results[0].Content)
	assert.Equal(t, float32(0.95), results[0].Score)
}

func TestStore_DeleteChunksBySourceID(t *testing.T) {
	client, ts := mockWeaviate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/meta" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.19.0"}`))
			return
		}
		assert.Equal(t, "/v1/batch/objects", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	})
	defer ts.Close()

	store := adapter.NewStore(client)
	err := store.DeleteChunksBySourceID(context.Background(), "src1")
	assert.NoError(t, err)
}

func TestStore_GetChunksByURL(t *testing.T) {
	client, ts := mockWeaviate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/meta" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.19.0"}`))
			return
		}
		assert.Equal(t, "/v1/graphql", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"Get": map[string]interface{}{
					"DocumentChunk": []interface{}{
						map[string]interface{}{
							"content": "url content",
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	store := adapter.NewStore(client)
	results, err := store.GetChunksByURL(context.Background(), "http://u.rl")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "url content", results[0].Content)
}

func TestStore_CountChunks(t *testing.T) {
	client, ts := mockWeaviate(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/meta" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version": "1.19.0"}`))
			return
		}
		assert.Equal(t, "/v1/graphql", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"Aggregate": map[string]interface{}{
					"DocumentChunk": []interface{}{
						map[string]interface{}{
							"meta": map[string]interface{}{
								"count": 42.0,
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	store := adapter.NewStore(client)
	count, err := store.CountChunks(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 42, count)
}

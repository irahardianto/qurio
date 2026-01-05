package weaviate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"qurio/apps/backend/internal/worker"
)

// --- Helpers ---

func newMockWeaviateServer(t *testing.T, checkFunc func(r *http.Request, body map[string]interface{})) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&body)
		}
		// Ignore startup checks
		if r.URL.Path == "/v1/meta" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version": "1.19.0",
			})
			return
		}
		if r.URL.Path == "/v1/.well-known/live" || r.URL.Path == "/v1/.well-known/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if checkFunc != nil {
			checkFunc(r, body)
		}

		// Mock responses based on path
		if r.URL.Path == "/v1/graphql" {
			// Return mock search result
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"Get": map[string]interface{}{
						"DocumentChunk": []interface{}{
							map[string]interface{}{
								"content": "hello world",
								"sourceId": "src-1",
								"_additional": map[string]interface{}{
									"score": "0.95",
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		
		if r.URL.Path == "/v1/objects" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"class": "DocumentChunk",
				"id":    "123",
			})
			return
		}

		if r.URL.Path == "/v1/batch/objects" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{}) // Batch delete returns object
			return
		}
	}))
}

func newTestStore(t *testing.T, server *httptest.Server) *Store {
	cfg := weaviate.Config{
		Host:   server.URL[7:], // Strip http://
		Scheme: "http",
	}
	client, err := weaviate.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return NewStore(client)
}

// --- Tests ---

func TestStore_StoreChunk(t *testing.T) {
	server := newMockWeaviateServer(t, func(r *http.Request, body map[string]interface{}) {
		assert.Equal(t, "/v1/objects", r.URL.Path)
		assert.Equal(t, "DocumentChunk", body["class"])
		props := body["properties"].(map[string]interface{})
		assert.Equal(t, "hello", props["content"])
		assert.Equal(t, "src-1", props["sourceId"])
	})
	defer server.Close()

	store := newTestStore(t, server)
	
	err := store.StoreChunk(context.Background(), worker.Chunk{
		Content: "hello",
		SourceID: "src-1",
	})
	assert.NoError(t, err)
}

func TestStore_Search(t *testing.T) {
	server := newMockWeaviateServer(t, func(r *http.Request, body map[string]interface{}) {
		assert.Equal(t, "/v1/graphql", r.URL.Path)
		query := body["query"].(string)
		// Relaxed checks
		assert.Contains(t, query, "Get")
		assert.Contains(t, query, "DocumentChunk")
		assert.Contains(t, query, "hybrid")
	})
	defer server.Close()

	store := newTestStore(t, server)

	results, err := store.Search(context.Background(), "test", nil, 0.5, 10, nil)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "hello world", results[0].Content)
}

func TestStore_DeleteChunksBySourceID(t *testing.T) {
	server := newMockWeaviateServer(t, func(r *http.Request, body map[string]interface{}) {
		assert.Equal(t, "/v1/batch/objects", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		match := body["match"].(map[string]interface{})
		assert.Equal(t, "DocumentChunk", match["class"])
		where := match["where"].(map[string]interface{})
		assert.Equal(t, "sourceId", where["path"].([]interface{})[0])
	})
	defer server.Close()

	store := newTestStore(t, server)

	err := store.DeleteChunksBySourceID(context.Background(), "src-1")
	assert.NoError(t, err)
}
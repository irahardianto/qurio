package weaviate_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	weaviateclient "github.com/weaviate/weaviate-go-client/v5/weaviate"
	adapter "qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/worker"
)

func TestStore_StoreChunk(t *testing.T) {
	// Mock Weaviate Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check URL for creation: /v1/objects
		if r.URL.Path == "/v1/objects" && r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"class": "DocumentChunk",
				"id":    "123",
			})
			return
		}
		http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
	}))
	defer ts.Close()

	// Config
	cfg := weaviateclient.Config{
		Host:   ts.Listener.Addr().String(),
		Scheme: "http",
	}
	client, err := weaviateclient.NewClient(cfg)
	assert.NoError(t, err)

	store := adapter.NewStore(client)

	ctx := context.Background()
	chunk := worker.Chunk{
		Content:    "test content",
		SourceURL:  "http://example.com",
		SourceID:   "src1",
		ChunkIndex: 0,
		Vector:     []float32{0.1, 0.2},
	}

	err = store.StoreChunk(ctx, chunk)
	assert.NoError(t, err)
}

func TestStore_Search(t *testing.T) {
	// Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/graphql" {
			w.WriteHeader(http.StatusOK)
			// Return mocked GraphQL response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"Get": map[string]interface{}{
						"DocumentChunk": []map[string]interface{}{
							{
								"content": "found content",
								"url":     "http://example.com",
								"_additional": map[string]interface{}{
									"score": 0.9,
								},
							},
						},
					},
				},
			})
			return
		}
		http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
	}))
	defer ts.Close()

	cfg := weaviateclient.Config{
		Host:   ts.Listener.Addr().String(),
		Scheme: "http",
	}
	client, err := weaviateclient.NewClient(cfg)
	assert.NoError(t, err)

	store := adapter.NewStore(client)

	results, err := store.Search(context.Background(), "query", []float32{0.1}, 0.5, 1, nil)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "found content", results[0].Content)
}

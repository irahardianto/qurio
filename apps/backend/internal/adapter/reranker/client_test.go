package reranker_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/adapter/reranker"
)

func TestClient_Rerank_Jina(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/rerank", r.URL.Path)
		assert.Equal(t, "Bearer k1", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"index": 1, "relevance_score": 0.9},
				{"index": 0, "relevance_score": 0.8},
			},
		})
	}))
	defer ts.Close()

	client := reranker.NewClient("jina", "k1")
	client.SetBaseURL(ts.URL + "/v1/rerank")

	indices, err := client.Rerank(context.Background(), "q", []string{"d1", "d2"})
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 0}, indices)
}

func TestClient_Rerank_Cohere(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/rerank", r.URL.Path)
		assert.Equal(t, "Bearer k2", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"index": 1, "relevance_score": 0.9},
				{"index": 0, "relevance_score": 0.8},
			},
		})
	}))
	defer ts.Close()

	client := reranker.NewClient("cohere", "k2")
	client.SetBaseURL(ts.URL + "/v1/rerank")

	indices, err := client.Rerank(context.Background(), "q", []string{"d1", "d2"})
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 0}, indices)
}

func TestClient_Rerank_None(t *testing.T) {
	client := reranker.NewClient("none", "")
	indices, err := client.Rerank(context.Background(), "q", []string{"d1", "d2"})
	assert.NoError(t, err)
	assert.Equal(t, []int{0, 1}, indices)
}

func TestClient_Rerank_ErrorHandling(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"detail":"invalid query"}`))
	}))
	defer ts.Close()

	client := reranker.NewClient("jina", "k1")
	client.SetBaseURL(ts.URL)

	_, err := client.Rerank(context.Background(), "q", []string{"d1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jina api error: 400")
	assert.Contains(t, err.Error(), `{"detail":"invalid query"}`)
}

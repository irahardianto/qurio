package vector_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate/entities/models"
	"qurio/apps/backend/internal/vector"
)

func TestWeaviateClientAdapter_ClassExists(t *testing.T) {
	t.Run("Exists", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/meta" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"version": "1.19.0"}`))
				return
			}
			assert.Equal(t, "/v1/schema/TestClass", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(&models.Class{Class: "TestClass"})
		}))
		defer ts.Close()

		cfg := weaviate.Config{Host: ts.Listener.Addr().String(), Scheme: "http"}
		client, _ := weaviate.NewClient(cfg)
		adapter := vector.NewWeaviateClientAdapter(client)

		exists, err := adapter.ClassExists(context.Background(), "TestClass")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NotFound", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/meta" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"version": "1.19.0"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		cfg := weaviate.Config{Host: ts.Listener.Addr().String(), Scheme: "http"}
		client, _ := weaviate.NewClient(cfg)
		adapter := vector.NewWeaviateClientAdapter(client)

		exists, err := adapter.ClassExists(context.Background(), "TestClass")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestWeaviateClientAdapter_CreateClass(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/meta" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"version": "1.19.0"}`))
				return
			}
			assert.Equal(t, "/v1/schema", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		cfg := weaviate.Config{Host: ts.Listener.Addr().String(), Scheme: "http"}
		client, _ := weaviate.NewClient(cfg)
		adapter := vector.NewWeaviateClientAdapter(client)

		err := adapter.CreateClass(context.Background(), &models.Class{Class: "NewClass"})
		assert.NoError(t, err)
	})
}

package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nsqio/go-nsq"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/internal/middleware"
)

// Mocks
type MockEmbedder struct {
	LastCtx  context.Context
	LastText string
}
func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	m.LastCtx = ctx
	m.LastText = text
	return []float32{0.1, 0.2}, nil
}

type MockStore struct {
	LastCtx   context.Context
	LastChunk Chunk
}
func (m *MockStore) StoreChunk(ctx context.Context, chunk Chunk) error {
	m.LastCtx = ctx
	m.LastChunk = chunk
	return nil
}
func (m *MockStore) DeleteChunksByURL(ctx context.Context, sourceID, url string) error {
	return nil
}
func (m *MockStore) CountChunks(ctx context.Context) (int, error) { return 0, nil }

type MockUpdater struct{}
func (m *MockUpdater) UpdateStatus(ctx context.Context, id, status string) error { return nil }
func (m *MockUpdater) UpdateBodyHash(ctx context.Context, id, hash string) error { return nil }

type MockJobRepo struct{}
func (m *MockJobRepo) Save(ctx context.Context, job *job.Job) error { return nil }
func (m *MockJobRepo) List(ctx context.Context) ([]job.Job, error) { return nil, nil }
func (m *MockJobRepo) Get(ctx context.Context, id string) (*job.Job, error) { return nil, nil }
func (m *MockJobRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *MockJobRepo) Count(ctx context.Context) (int, error) { return 0, nil }

type MockSourceFetcher struct{}
func (m *MockSourceFetcher) GetSourceDetails(ctx context.Context, id string) (string, string, error) { return "web", "http://example.com", nil }
func (m *MockSourceFetcher) GetSourceConfig(ctx context.Context, id string) (int, []string, string, string, error) { return 0, nil, "", "test-source", nil }

type MockPageManager struct{}
func (m *MockPageManager) BulkCreatePages(ctx context.Context, pages []PageDTO) ([]string, error) { return nil, nil }
func (m *MockPageManager) UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error { return nil }
func (m *MockPageManager) CountPendingPages(ctx context.Context, sourceID string) (int, error) { return 0, nil }

type MockPublisher struct{}
func (m *MockPublisher) Publish(topic string, body []byte) error { return nil }

func TestResultConsumer_HandleMessage_CorrelationID(t *testing.T) {
	embedder := &MockEmbedder{}
	store := &MockStore{}
	consumer := NewResultConsumer(embedder, store, &MockUpdater{}, &MockJobRepo{}, &MockSourceFetcher{}, &MockPageManager{}, &MockPublisher{})

	expectedID := "test-correlation-id"
	payload := map[string]string{
		"source_id": "src-1",
		"content": "test content",
		"url": "http://example.com",
		"status": "success",
		"correlation_id": expectedID,
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	if err := consumer.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Check Embedder Context
	if embedder.LastCtx == nil {
		t.Fatal("Embedder not called")
	}
	if id := middleware.GetCorrelationID(embedder.LastCtx); id != expectedID {
		t.Errorf("Embedder context missing correlation ID. Got '%s', expected '%s'", id, expectedID)
	}

	// Check Store Context
	if store.LastCtx == nil {
		t.Fatal("Store not called")
	}
	if id := middleware.GetCorrelationID(store.LastCtx); id != expectedID {
		t.Errorf("Store context missing correlation ID. Got '%s', expected '%s'", id, expectedID)
	}
}

func TestResultConsumer_PopulatesSourceName(t *testing.T) {
	embedder := &MockEmbedder{}
	store := &MockStore{}
	consumer := NewResultConsumer(embedder, store, &MockUpdater{}, &MockJobRepo{}, &MockSourceFetcher{}, &MockPageManager{}, &MockPublisher{})

	payload := map[string]string{
		"source_id": "src-1",
		"content":   "test content",
		"url":       "http://example.com",
		"status":    "success",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	if err := consumer.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	if store.LastChunk.SourceName != "test-source" {
		t.Errorf("Expected SourceName 'test-source', got '%s'", store.LastChunk.SourceName)
	}
}

func TestHandleMessage_WithMetadata(t *testing.T) {
	embedder := &MockEmbedder{}
	store := &MockStore{}
	consumer := NewResultConsumer(embedder, store, &MockUpdater{}, &MockJobRepo{}, &MockSourceFetcher{}, &MockPageManager{}, &MockPublisher{})

	payload := map[string]interface{}{
		"source_id": "src-1",
		"content":   "test content",
		"url":       "http://example.com",
		"status":    "success",
		"metadata": map[string]interface{}{
			"author":     "John Doe",
			"created_at": "2023-01-01",
			"pages":      10,
		},
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	if err := consumer.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify Author is in the embedded text
	if !contains(embedder.LastText, "Author: John Doe") {
		t.Errorf("Embedded text missing Author. Got: %s", embedder.LastText)
	}
	// Verify Created is in the embedded text
	if !contains(embedder.LastText, "Created: 2023-01-01") {
		t.Errorf("Embedded text missing Created. Got: %s", embedder.LastText)
	}

	// Verify Chunk metadata
	if store.LastChunk.Author != "John Doe" {
		t.Errorf("Chunk Author mismatch. Got: %s, Want: John Doe", store.LastChunk.Author)
	}
	if store.LastChunk.CreatedAt != "2023-01-01" {
		t.Errorf("Chunk CreatedAt mismatch. Got: %s, Want: 2023-01-01", store.LastChunk.CreatedAt)
	}
	if store.LastChunk.PageCount != 10 {
		t.Errorf("Chunk PageCount mismatch. Got: %d, Want: 10", store.LastChunk.PageCount)
	}
}

func contains(s, substr string) bool {
    for i := 0; i < len(s)-len(substr)+1; i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}

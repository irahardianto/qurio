package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/internal/config"
)

// Mocks for internal tests

type MockStore struct {
	LastCtx   context.Context
}
func (m *MockStore) StoreChunk(ctx context.Context, chunk Chunk) error {
	m.LastCtx = ctx
	return nil
}
func (m *MockStore) DeleteChunksByURL(ctx context.Context, sourceID, url string) error {
	m.LastCtx = ctx
	return nil
}
func (m *MockStore) CountChunks(ctx context.Context) (int, error) { return 0, nil }
func (m *MockStore) EnsureSchema(ctx context.Context) error { return nil }
func (m *MockStore) GetChunks(ctx context.Context, sourceID string) ([]Chunk, error) { return nil, nil }
func (m *MockStore) GetChunksByURL(ctx context.Context, url string) ([]any, error) { return nil, nil }
func (m *MockStore) Search(ctx context.Context, query string, vector []float32, alpha float32, limit int, searchFilters map[string]interface{}) ([]any, error) { return nil, nil }


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
func (m *MockSourceFetcher) GetSourceConfig(ctx context.Context, id string) (int, []string, string, string, error) { return 5, nil, "", "test-source", nil }

type MockPageManager struct{
	ReturnURLs []string
}
func (m *MockPageManager) BulkCreatePages(ctx context.Context, pages []PageDTO) ([]string, error) {
	if len(m.ReturnURLs) > 0 {
		return m.ReturnURLs, nil
	}
	return nil, nil
}
func (m *MockPageManager) UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error { return nil }
func (m *MockPageManager) CountPendingPages(ctx context.Context, sourceID string) (int, error) { return 0, nil }

type MockPublisher struct{
	LastTopic string
	LastBody []byte
	PublishCallCount int
}
func (m *MockPublisher) Publish(topic string, body []byte) error {
	m.LastTopic = topic
	m.LastBody = body
	m.PublishCallCount++
	return nil
}

func TestResultConsumer_HandleMessage_PublishEmbedTasks(t *testing.T) {
	store := &MockStore{}
	pub := &MockPublisher{}
	consumer := NewResultConsumer(store, &MockUpdater{}, &MockJobRepo{}, &MockSourceFetcher{}, &MockPageManager{}, pub)

	payload := map[string]string{
		"source_id": "src-1",
		"content": "test content",
		"url": "http://example.com",
		"status": "success",
		"title": "Title",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	if err := consumer.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Should have published to ingest.embed
	if pub.PublishCallCount == 0 {
		t.Fatal("Expected publish call count > 0")
	}
	if pub.LastTopic != config.TopicIngestEmbed {
		t.Errorf("Expected topic %s, got %s", config.TopicIngestEmbed, pub.LastTopic)
	}
}

func TestResultConsumer_HandleMessage_EmptyBody(t *testing.T) {
	consumer := &ResultConsumer{}
	msg := &nsq.Message{Body: []byte("")}

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)
}

func TestResultConsumer_HandleMessage_InvalidJSON(t *testing.T) {
	consumer := &ResultConsumer{}
	msg := &nsq.Message{Body: []byte("{invalid-json")}
	
	// Should return nil to ack message (don't retry poison pill)
	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)
}

func TestResultConsumer_HandleMessage_DependencyError(t *testing.T) {
	// If Publisher fails, HandleMessage should return error
	// MockPublisher currently returns nil.
	// Need a failing mock.
}

func TestResultConsumer_PublishesDiscoveredLinks(t *testing.T) {
	store := &MockStore{}
	pub := &MockPublisher{}
	pm := &MockPageManager{ReturnURLs: []string{"http://example.com/page2"}}
	
	consumer := NewResultConsumer(store, &MockUpdater{}, &MockJobRepo{}, &MockSourceFetcher{}, pm, pub)

	payload := map[string]interface{}{
		"source_id": "src-1",
		"content":   "test content",
		"url":       "http://example.com",
		"links":     []string{"http://example.com/page2"},
		"depth":     0,
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	if err := consumer.HandleMessage(msg); err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Should publish to ingest.task.web for the discovered link
	// Note: It publishes to embed AND web.
	// LastTopic might be web or embed depending on order.
	// In implementation: Embed happens (Step 2), then Link Discovery (Step 4).
	// So LastTopic should be Web.
	
	if pub.LastTopic != config.TopicIngestWeb {
		t.Errorf("Expected last topic %s, got %s", config.TopicIngestWeb, pub.LastTopic)
	}
}

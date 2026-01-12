package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
)

// Helper struct for this test
type TrackingMockPublisher struct {
    PublishCallCount int
}
func (m *TrackingMockPublisher) Publish(topic string, body []byte) error {
    m.PublishCallCount++
    return nil
}

type TestMockPageManager struct{}
func (m *TestMockPageManager) BulkCreatePages(ctx context.Context, pages []PageDTO) ([]string, error) {
    urls := make([]string, len(pages))
    for i, p := range pages {
        urls[i] = p.URL
    }
    return urls, nil
}
func (m *TestMockPageManager) UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error { return nil }
func (m *TestMockPageManager) CountPendingPages(ctx context.Context, sourceID string) (int, error) { return 0, nil }

func TestHandleMessage_LLMsTxt_BypassesDepth(t *testing.T) {
	// Arrange
    mockPub := &TrackingMockPublisher{}
    mockPageMgr := &TestMockPageManager{}
    
    consumer := NewResultConsumer(&MockStore{}, &MockUpdater{}, &MockJobRepo{}, &MockSourceFetcher{}, mockPageMgr, mockPub)

	payload := map[string]interface{}{
		"source_id":  "src-1",
		"url":        "https://example.com/llms.txt",
		"links":      []string{"https://example.com/found"},
		"depth":      0,
        // MaxDepth is fetched from SourceFetcher mock, which returns 0
        "status":     "success",
        "content":    "some content",
        "title":      "LLMs.txt",
        "path":       "/llms.txt",
	}
    
    body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	// Act
	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)

	// Assert
    // If logic is patched, 0 < 1 (virtual maxDepth), so discovery happens -> Message published.
	// Note: It publishes 1 embedding task + 1 web task = 2 tasks.
    assert.Equal(t, 2, mockPub.PublishCallCount, "Should publish 1 embedding task + 1 new web task for the discovered link")
}

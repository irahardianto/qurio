package worker_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/internal/worker"
)

// Mocks
type MockEmbedder struct { mock.Mock }
func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]float32), args.Error(1)
}

type MockVectorStore struct { mock.Mock }
func (m *MockVectorStore) StoreChunk(ctx context.Context, chunk worker.Chunk) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}
func (m *MockVectorStore) DeleteChunksByURL(ctx context.Context, sourceID, url string) error {
	args := m.Called(ctx, sourceID, url)
	return args.Error(0)
}

type MockUpdater struct { mock.Mock }
func (m *MockUpdater) UpdateStatus(ctx context.Context, id, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
func (m *MockUpdater) UpdateBodyHash(ctx context.Context, id, hash string) error {
	args := m.Called(ctx, id, hash)
	return args.Error(0)
}

type MockJobRepo struct { mock.Mock }
func (m *MockJobRepo) Save(ctx context.Context, j *job.Job) error {
	args := m.Called(ctx, j)
	return args.Error(0)
}
func (m *MockJobRepo) List(ctx context.Context) ([]job.Job, error) { return nil, nil }
func (m *MockJobRepo) Get(ctx context.Context, id string) (*job.Job, error) { return nil, nil }
func (m *MockJobRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *MockJobRepo) Count(ctx context.Context) (int, error) { return 0, nil }

type MockSourceFetcher struct { mock.Mock }
func (m *MockSourceFetcher) GetSourceConfig(ctx context.Context, id string) (int, []string, string, string, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Get(1).([]string), args.String(2), args.String(3), args.Error(4)
}
func (m *MockSourceFetcher) GetSourceDetails(ctx context.Context, id string) (string, string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.String(1), args.Error(2)
}

type MockPageManager struct { mock.Mock }
func (m *MockPageManager) BulkCreatePages(ctx context.Context, pages []worker.PageDTO) ([]string, error) {
	args := m.Called(ctx, pages)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockPageManager) UpdatePageStatus(ctx context.Context, sourceID, url, status, errStr string) error {
	args := m.Called(ctx, sourceID, url, status, errStr)
	return args.Error(0)
}
func (m *MockPageManager) CountPendingPages(ctx context.Context, sourceID string) (int, error) {
	args := m.Called(ctx, sourceID)
	return args.Int(0), args.Error(1)
}

type MockTaskPublisher struct { mock.Mock }
func (m *MockTaskPublisher) Publish(topic string, body []byte) error {
	args := m.Called(topic, body)
	return args.Error(0)
}

func TestHandleMessage_Success(t *testing.T) {
	// Setup Mocks
	s := new(MockVectorStore)
	u := new(MockUpdater)
	j := new(MockJobRepo)
	sf := new(MockSourceFetcher)
	pm := new(MockPageManager)
	tp := new(MockTaskPublisher)

	consumer := worker.NewResultConsumer(s, u, j, sf, pm, tp)

	// Payload
	payload := map[string]interface{}{
		"source_id": "src1",
		"url": "http://example.com",
		"content": "Some content",
		"title": "Title",
		"status": "success",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	// Expectations
	sf.On("GetSourceConfig", mock.Anything, "src1").Return(2, []string{}, "", "My Source", nil)
	s.On("DeleteChunksByURL", mock.Anything, "src1", "http://example.com").Return(nil)
	
	// Expect Publish instead of Embed/Store
	tp.On("Publish", "ingest.embed", mock.Anything).Return(nil)

	u.On("UpdateBodyHash", mock.Anything, "src1", mock.Anything).Return(nil)
	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com", "completed", "").Return(nil)
	pm.On("CountPendingPages", mock.Anything, "src1").Return(0, nil)
	u.On("UpdateStatus", mock.Anything, "src1", "completed").Return(nil)

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)
	
	s.AssertExpectations(t)
	pm.AssertExpectations(t)
	tp.AssertExpectations(t)
}

func TestHandleMessage_Failure(t *testing.T) {
	// Setup Mocks
	s := new(MockVectorStore)
	u := new(MockUpdater)
	j := new(MockJobRepo)
	sf := new(MockSourceFetcher)
	pm := new(MockPageManager)
	tp := new(MockTaskPublisher)

	consumer := worker.NewResultConsumer(s, u, j, sf, pm, tp)

	payload := map[string]interface{}{
		"source_id": "src1",
		"url": "http://example.com",
		"status": "failed",
		"error": "Some error",
		"depth": 1,
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com", "failed", "Some error").Return(nil)
	// Depth 1 -> No UpdateStatus(failed) for source
	// Save Failed Job? OriginalPayload is nil in this map, so maybe skipped?
	// Payload struct has OriginalPayload json.RawMessage.
	// If we provide it in map:
	// "original_payload": {}
	// Let's assume nil original payload for simplicity
	
	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)
	
	pm.AssertExpectations(t)
}

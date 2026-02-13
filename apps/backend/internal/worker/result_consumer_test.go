package worker_test

import (
	"encoding/json"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/worker"
)

func TestResultConsumer_HandleMessage_Success(t *testing.T) {
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
		"url":       "http://example.com",
		"content":   "Some content",
		"title":     "Title",
		"status":    "success",
		"links":     []string{"http://example.com/subpage"},
		"depth":     0,
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	// Expectations
	// 1. Fetch Config
	sf.On("GetSourceConfig", mock.Anything, "src1").Return(2, []string{}, "api-key", "My Source", nil)

	// 2. Delete Old Chunks
	s.On("DeleteChunksByURL", mock.Anything, "src1", "http://example.com").Return(nil)

	// 3. Publish Embed Tasks (Chunking happens internally)
	tp.On("Publish", config.TopicIngestEmbed, mock.MatchedBy(func(b []byte) bool {
		var p worker.IngestEmbedPayload
		json.Unmarshal(b, &p)
		return p.SourceID == "src1" && p.SourceName == "My Source" && p.Content == "Some content"
	})).Return(nil)

	// 4. Update Body Hash
	u.On("UpdateBodyHash", mock.Anything, "src1", mock.Anything).Return(nil)

	// 5. Link Discovery -> Publish Web Task
	pm.On("BulkCreatePages", mock.Anything, mock.MatchedBy(func(pages []worker.PageDTO) bool {
		return len(pages) == 1 && pages[0].URL == "http://example.com/subpage"
	})).Return([]string{"http://example.com/subpage"}, nil)

	tp.On("Publish", config.TopicIngestWeb, mock.MatchedBy(func(b []byte) bool {
		var p map[string]interface{}
		json.Unmarshal(b, &p)
		return p["url"] == "http://example.com/subpage" && p["depth"] == float64(1)
	})).Return(nil)

	// 6. Update Page Status (Completed)
	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com", "completed", "").Return(nil)

	// 7. Check Source Completion
	pm.On("CountPendingPages", mock.Anything, "src1").Return(0, nil)
	u.On("UpdateStatus", mock.Anything, "src1", "completed").Return(nil)

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)

	s.AssertExpectations(t)
	pm.AssertExpectations(t)
	tp.AssertExpectations(t)
	u.AssertExpectations(t)
}

func TestResultConsumer_HandleMessage_Failure(t *testing.T) {
	// Setup Mocks
	s := new(MockVectorStore)
	u := new(MockUpdater)
	j := new(MockJobRepo)
	sf := new(MockSourceFetcher)
	pm := new(MockPageManager)
	tp := new(MockTaskPublisher)

	consumer := worker.NewResultConsumer(s, u, j, sf, pm, tp)

	originalPayload := map[string]interface{}{"foo": "bar"}

	payload := map[string]interface{}{
		"source_id":        "src1",
		"url":              "http://example.com",
		"status":           "failed",
		"error":            "Some error",
		"depth":            0,
		"original_payload": originalPayload,
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	// Expectations
	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com", "failed", "Some error").Return(nil)
	u.On("UpdateStatus", mock.Anything, "src1", "failed").Return(nil) // Depth 0 -> Update Source Status
	j.On("Save", mock.Anything, mock.MatchedBy(func(job *job.Job) bool {
		return job.SourceID == "src1" && job.Error == "Some error"
	})).Return(nil)

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err) // Should not error, handled gracefully

	pm.AssertExpectations(t)
	u.AssertExpectations(t)
	j.AssertExpectations(t)
}

func TestResultConsumer_HandleMessage_LLMsTxt_ExtendedDepth(t *testing.T) {
	s := new(MockVectorStore)
	u := new(MockUpdater)
	j := new(MockJobRepo)
	sf := new(MockSourceFetcher)
	pm := new(MockPageManager)
	tp := new(MockTaskPublisher)

	consumer := worker.NewResultConsumer(s, u, j, sf, pm, tp)

	payload := map[string]interface{}{
		"source_id": "src1",
		"url":       "http://example.com/llms.txt",
		"content":   "content",
		"links":     []string{"http://example.com/doc.md"},
		"depth":     2,
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	// Mock Config: Max Depth is 2.
	// Normal logic: Depth 2 == Max Depth 2 -> No new links.
	// LLMs.txt logic: Effective Max Depth = 3. -> New links allowed.
	sf.On("GetSourceConfig", mock.Anything, "src1").Return(2, []string{}, "", "Src", nil)
	s.On("DeleteChunksByURL", mock.Anything, "src1", "http://example.com/llms.txt").Return(nil)
	tp.On("Publish", config.TopicIngestEmbed, mock.Anything).Return(nil)
	u.On("UpdateBodyHash", mock.Anything, "src1", mock.Anything).Return(nil)

	// Expect Link Discovery
	pm.On("BulkCreatePages", mock.Anything, mock.MatchedBy(func(pages []worker.PageDTO) bool {
		return len(pages) == 1 && pages[0].URL == "http://example.com/doc.md"
	})).Return([]string{"http://example.com/doc.md"}, nil)

	tp.On("Publish", config.TopicIngestWeb, mock.MatchedBy(func(b []byte) bool {
		var p map[string]interface{}
		json.Unmarshal(b, &p)
		return p["depth"] == float64(3) // 2 + 1
	})).Return(nil)

	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com/llms.txt", "completed", "").Return(nil)
	pm.On("CountPendingPages", mock.Anything, "src1").Return(1, nil) // Still pending pages

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)

	tp.AssertExpectations(t)
}

func TestResultConsumer_HandleMessage_DeleteChunksError(t *testing.T) {
	s := new(MockVectorStore)
	// other mocks...
	sf := new(MockSourceFetcher)

	consumer := worker.NewResultConsumer(s, nil, nil, sf, nil, nil)

	payload := map[string]interface{}{
		"source_id": "src1",
		"url":       "http://example.com",
		"status":    "success",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	sf.On("GetSourceConfig", mock.Anything, "src1").Return(2, []string{}, "", "Src", nil)
	s.On("DeleteChunksByURL", mock.Anything, "src1", "http://example.com").Return(assert.AnError)

	err := consumer.HandleMessage(msg)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestResultConsumer_HandleMessage_PoisonPill(t *testing.T) {
	consumer := worker.NewResultConsumer(nil, nil, nil, nil, nil, nil)
	msg := &nsq.Message{Body: []byte("invalid json")}

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)
}

func TestResultConsumer_HandleMessage_MissingRequiredFields(t *testing.T) {
	consumer := worker.NewResultConsumer(nil, nil, nil, nil, nil, nil)

	// Missing URL
	payload := map[string]interface{}{"source_id": "src1"}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)
}

func TestResultConsumer_HandleMessage_LinkDiscoveryPublishFailure(t *testing.T) {
	s := new(MockVectorStore)
	sf := new(MockSourceFetcher)
	pm := new(MockPageManager)
	tp := new(MockTaskPublisher)
	u := new(MockUpdater)

	consumer := worker.NewResultConsumer(s, u, nil, sf, pm, tp)

	payload := map[string]interface{}{
		"source_id": "src1",
		"url":       "http://example.com",
		"content":   "c",
		"links":     []string{"http://example.com/sub"},
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	sf.On("GetSourceConfig", mock.Anything, "src1").Return(5, []string{}, "", "Src", nil)
	s.On("DeleteChunksByURL", mock.Anything, "src1", "http://example.com").Return(nil)
	tp.On("Publish", config.TopicIngestEmbed, mock.Anything).Return(nil)
	u.On("UpdateBodyHash", mock.Anything, "src1", mock.Anything).Return(nil)

	// Link Discovery Success
	pm.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{"http://example.com/sub"}, nil)

	// Publish Web Task Failure
	tp.On("Publish", config.TopicIngestWeb, mock.Anything).Return(assert.AnError)

	// Should mark new page as failed
	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com/sub", "failed", mock.Anything).Return(nil)

	// Should still complete the current page
	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com", "completed", "").Return(nil)
	pm.On("CountPendingPages", mock.Anything, "src1").Return(0, nil)
	u.On("UpdateStatus", mock.Anything, "src1", "completed").Return(nil)

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err) // Non-fatal for the consumer

	tp.AssertExpectations(t)
	pm.AssertExpectations(t)
}

func TestResultConsumer_HandleMessage_EmptyContent_NoEmbedPublish(t *testing.T) {
	s := new(MockVectorStore)
	sf := new(MockSourceFetcher)
	u := new(MockUpdater)
	pm := new(MockPageManager)

	consumer := worker.NewResultConsumer(s, u, nil, sf, pm, nil)

	payload := map[string]interface{}{
		"source_id": "src1",
		"url":       "http://example.com",
		"content":   "", // Empty
		"status":    "success",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	sf.On("GetSourceConfig", mock.Anything, "src1").Return(5, []string{}, "", "Src", nil)
	s.On("DeleteChunksByURL", mock.Anything, "src1", "http://example.com").Return(nil)
	u.On("UpdateBodyHash", mock.Anything, "src1", mock.Anything).Return(nil)
	pm.On("UpdatePageStatus", mock.Anything, "src1", "http://example.com", "completed", "").Return(nil)
	pm.On("CountPendingPages", mock.Anything, "src1").Return(1, nil)

	// Expect NO calls to publisher

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)
}

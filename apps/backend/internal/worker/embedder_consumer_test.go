package worker_test

import (
	"encoding/json"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/internal/worker"
)

func TestEmbedderConsumer_HandleMessage_Success(t *testing.T) {
	e := new(MockEmbedder)
	s := new(MockVectorStore)

	consumer := worker.NewEmbedderConsumer(e, s)

	payload := worker.IngestEmbedPayload{
		SourceID:   "src1",
		SourceURL:  "http://example.com",
		Content:    "Chunk Content",
		ChunkIndex: 0,
		Title:      "Title",
		Author:     "John Doe",
		CreatedAt:  "2023-01-01",
		ChunkType:  "text",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	// Expect Embed call with formatted context string
	e.On("Embed", mock.Anything, mock.MatchedBy(func(text string) bool {
		// Check that metadata is included in embedding context
		return assert.Contains(t, text, "Title: Title") &&
			assert.Contains(t, text, "Author: John Doe") &&
			assert.Contains(t, text, "Created: 2023-01-01") &&
			assert.Contains(t, text, "Chunk Content")
	})).Return([]float32{0.1, 0.2}, nil)

	// Expect Store call
	s.On("StoreChunk", mock.Anything, mock.MatchedBy(func(c worker.Chunk) bool {
		return c.SourceID == "src1" &&
			c.Author == "John Doe" &&
			c.Vector[0] == 0.1
	})).Return(nil)

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	s.AssertExpectations(t)
}

func TestEmbedderConsumer_HandleMessage_EmbedError(t *testing.T) {
	e := new(MockEmbedder)
	s := new(MockVectorStore)
	consumer := worker.NewEmbedderConsumer(e, s)

	payload := worker.IngestEmbedPayload{
		SourceID: "src1",
		Content:  "content",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	e.On("Embed", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	err := consumer.HandleMessage(msg)
	assert.Error(t, err) // Should retry
	assert.Equal(t, assert.AnError, err)
}

func TestEmbedderConsumer_HandleMessage_StoreError(t *testing.T) {
	e := new(MockEmbedder)
	s := new(MockVectorStore)
	consumer := worker.NewEmbedderConsumer(e, s)

	payload := worker.IngestEmbedPayload{
		SourceID: "src1",
		Content:  "content",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	e.On("Embed", mock.Anything, mock.Anything).Return([]float32{0.1}, nil)
	s.On("StoreChunk", mock.Anything, mock.Anything).Return(assert.AnError)

	err := consumer.HandleMessage(msg)
	assert.Error(t, err) // Should retry
}

func TestEmbedderConsumer_HandleMessage_PoisonPill(t *testing.T) {
	consumer := worker.NewEmbedderConsumer(nil, nil)
	msg := &nsq.Message{Body: []byte("invalid json")}

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err) // No retry
}

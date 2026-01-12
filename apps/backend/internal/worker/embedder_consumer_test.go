package worker_test

import (
	"encoding/json"
	"testing"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/internal/worker"
)

func TestEmbedderConsumer_HandleMessage(t *testing.T) {
	// Setup Mocks
	e := new(MockEmbedder)
	s := new(MockVectorStore)
	
	consumer := worker.NewEmbedderConsumer(e, s)

	// Payload
	payload := worker.IngestEmbedPayload{
		SourceID:   "src1",
		SourceURL:  "http://example.com",
		Content:    "Test content",
		ChunkIndex: 0,
		Title:      "Test Title",
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{Body: body}

	// Expectations
	e.On("Embed", mock.Anything, mock.MatchedBy(func(text string) bool {
		return assert.Contains(t, text, "Test content") && assert.Contains(t, text, "Test Title")
	})).Return([]float32{0.1, 0.2}, nil)
	
	s.On("StoreChunk", mock.Anything, mock.MatchedBy(func(chunk worker.Chunk) bool {
		return chunk.SourceID == "src1" && chunk.ChunkIndex == 0
	})).Return(nil)

	// Execute
	err := consumer.HandleMessage(msg)
	
	// Assert
	assert.NoError(t, err)
	e.AssertExpectations(t)
	s.AssertExpectations(t)
}

func TestEmbedderConsumer_PoisonPill(t *testing.T) {
	e := new(MockEmbedder)
	s := new(MockVectorStore)
	consumer := worker.NewEmbedderConsumer(e, s)

	msg := &nsq.Message{Body: []byte("invalid json")}

	err := consumer.HandleMessage(msg)
	assert.NoError(t, err) // Should return nil (ack)
}

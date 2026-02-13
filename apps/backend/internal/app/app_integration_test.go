package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	weaviate_adapter "qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/testutils"
	"qurio/apps/backend/internal/worker"
)

// MockEmbedder for E2E
type MockE2EEmbedder struct {
	mock.Mock
}

func (m *MockE2EEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float32), args.Error(1)
}

func TestApp_EndToEnd_Ingestion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E integration test")
	}

	// 1. Setup Infrastructure
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	logger := s.Logger()
	cfg := s.GetAppConfig()
	cfg.EnableEmbedderWorker = true
	cfg.GeminiAPIKey = "test-key"

	// 2. Setup Mocks
	mockEmbedder := new(MockE2EEmbedder)
	// Expect Embed call
	mockEmbedder.On("Embed", mock.Anything, mock.Anything).Return([]float32{0.1, 0.2, 0.3}, nil)

	// 3. Initialize App
	vecStore := weaviate_adapter.NewStore(s.Weaviate)
	require.NoError(t, vecStore.EnsureSchema(context.Background()))

	opts := &app.Options{
		Embedder: mockEmbedder,
	}

	application, err := app.New(cfg, s.DB, vecStore, s.NSQ, logger, opts)
	require.NoError(t, err)

	// 4. Create Source via HTTP
	createPayload := map[string]interface{}{
		"type": "web",
		"url":  "http://example.com/e2e",
		"name": "E2E Source",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/sources", bytes.NewReader(body))
	w := httptest.NewRecorder()

	application.Handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code) // SourceHandler.Create returns 201

	// Wait for NSQ message on 'ingest.task.web'
	webMsg := s.ConsumeOne(config.TopicIngestWeb)
	require.NotNil(t, webMsg, "Should receive web task")

	var taskPayload map[string]interface{}
	err = json.Unmarshal(webMsg.Body, &taskPayload)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/e2e", taskPayload["url"])
	sourceID := taskPayload["id"].(string)

	// 5. Simulate Worker Result (Crawler Success)
	resultPayload := map[string]interface{}{
		"source_id": sourceID,
		"url":       "http://example.com/e2e",
		"content":   "This is the content of the page for E2E testing.",
		"title":     "E2E Page",
		"status":    "success",
		"depth":     0,
	}
	resultBody, _ := json.Marshal(resultPayload)

	msg := &nsq.Message{
		Body: resultBody,
		ID:   nsq.MessageID{'1'},
	}

	// Execute Result Consumer Logic
	err = application.ResultConsumer.HandleMessage(msg)
	require.NoError(t, err)

	// 6. Verify Embed Task Published
	embedMsg := s.ConsumeOne(config.TopicIngestEmbed)
	require.NotNil(t, embedMsg, "Should receive embed task")

	var embedPayload worker.IngestEmbedPayload
	err = json.Unmarshal(embedMsg.Body, &embedPayload)
	require.NoError(t, err)
	assert.Equal(t, sourceID, embedPayload.SourceID)
	assert.Contains(t, embedPayload.Content, "This is the content")

	// 7. Simulate Embed Worker
	embedNsqMsg := &nsq.Message{
		Body: embedMsg.Body,
		ID:   nsq.MessageID{'2'},
	}

	err = application.EmbedderConsumer.HandleMessage(embedNsqMsg)
	require.NoError(t, err)

	// 8. Verify Weaviate Storage
	time.Sleep(1 * time.Second)

	details, err := application.SourceService.Get(context.Background(), sourceID, 10, 0, true)
	require.NoError(t, err)
	assert.NotEmpty(t, details.Chunks)
	assert.Equal(t, "E2E Page", details.Chunks[0].Title)

	// 9. Verify Mock usage
	mockEmbedder.AssertExpectations(t)
}

package worker_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/testutils"
	"qurio/apps/backend/internal/worker"
)

type MockChunkStore struct{}

func (m *MockChunkStore) GetChunks(ctx context.Context, sourceID string, limit, offset int) ([]worker.Chunk, error) {
	return nil, nil
}

func (m *MockChunkStore) DeleteChunksBySourceID(ctx context.Context, sourceID string) error {
	return nil
}

func (m *MockChunkStore) CountChunksBySource(ctx context.Context, sourceID string) (int, error) {
	return 0, nil
}

type MockSettings struct{}

func (m *MockSettings) Get(ctx context.Context) (*settings.Settings, error) { return nil, nil }

func TestTopicRouting(t *testing.T) {
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	ctx := context.Background()

	// 1. Setup Service
	repo := source.NewPostgresRepo(s.DB)
	svc := source.NewService(repo, s.NSQ, &MockChunkStore{}, &MockSettings{})

	// 2. Setup Consumers for verification
	webChan := make(chan *nsq.Message, 1)
	fileChan := make(chan *nsq.Message, 1)

	nsqCfg := nsq.NewConfig()

	// Web Consumer
	webConsumer, err := nsq.NewConsumer(config.TopicIngestWeb, "test-ch-web", nsqCfg)
	require.NoError(t, err)
	webConsumer.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		webChan <- m
		return nil
	}))

	appCfg := s.GetAppConfig()
	if err := webConsumer.ConnectToNSQD(appCfg.NSQDHost); err != nil {
		t.Fatalf("Failed to connect to NSQD: %v", err)
	}

	// File Consumer
	fileConsumer, err := nsq.NewConsumer(config.TopicIngestFile, "test-ch-file", nsqCfg)
	require.NoError(t, err)
	fileConsumer.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		fileChan <- m
		return nil
	}))
	if err := fileConsumer.ConnectToNSQD(appCfg.NSQDHost); err != nil {
		t.Fatalf("Failed to connect to NSQD: %v", err)
	}

	// 3. Action: Create Web Source
	webSrc := &source.Source{Type: "web", URL: "http://example.com/topic-test"}
	err = svc.Create(ctx, webSrc)
	require.NoError(t, err)

	// 4. Verify Web Topic
	select {
	case msg := <-webChan:
		var payload map[string]interface{}
		json.Unmarshal(msg.Body, &payload)
		assert.Equal(t, "web", payload["type"])
		assert.Equal(t, "http://example.com/topic-test", payload["url"])
		msg.Finish()
	case <-time.After(10 * time.Second): // Increase timeout for integration test
		t.Fatal("Timeout waiting for web task")
	}

	// 5. Action: Create File Source
	// Upload calls repo.Save then Publish
	_, err = svc.Upload(ctx, "/tmp/test.pdf", "hash-topic-test", "Test PDF")
	require.NoError(t, err)

	// 6. Verify File Topic
	select {
	case msg := <-fileChan:
		var payload map[string]interface{}
		json.Unmarshal(msg.Body, &payload)
		assert.Equal(t, "file", payload["type"])
		assert.Equal(t, "/tmp/test.pdf", payload["path"])
		msg.Finish()
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for file task")
	}
}

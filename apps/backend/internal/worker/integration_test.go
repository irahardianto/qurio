package worker_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/features/job"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/adapter/weaviate"
	"qurio/apps/backend/internal/testutils"
	"qurio/apps/backend/internal/worker"
)

// MockEmbedder for integration test (we don't hit real Gemini)
type MockEmbedder struct {
	mock.Mock
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Return a dummy vector
	return []float32{0.1, 0.2, 0.3}, nil
}

// TestSourceFetcher adapts source.Repository to worker.SourceFetcher
type TestSourceFetcher struct {
	Repo *source.PostgresRepo
}

func (f *TestSourceFetcher) GetSourceConfig(ctx context.Context, id string) (int, []string, string, string, error) {
	src, err := f.Repo.Get(ctx, id)
	if err != nil {
		return 0, nil, "", "", err
	}
	// Return default API Key and Source Name
	return src.MaxDepth, src.Exclusions, "dummy-api-key", src.Name, nil
}

func (f *TestSourceFetcher) GetSourceDetails(ctx context.Context, id string) (string, string, error) {
	src, err := f.Repo.Get(ctx, id)
	if err != nil {
		return "", "", err
	}
	return src.URL, src.Name, nil
}

type PageManagerAdapter struct {
	Repo *source.PostgresRepo
}

func (a *PageManagerAdapter) BulkCreatePages(ctx context.Context, pages []worker.PageDTO) ([]string, error) {
	srcPages := make([]source.SourcePage, len(pages))
	for i, p := range pages {
		srcPages[i] = source.SourcePage{
			SourceID: p.SourceID,
			URL:      p.URL,
			Status:   p.Status,
			Depth:    p.Depth,
		}
	}
	return a.Repo.BulkCreatePages(ctx, srcPages)
}

func (a *PageManagerAdapter) UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error {
	return a.Repo.UpdatePageStatus(ctx, sourceID, url, status, err)
}

func (a *PageManagerAdapter) CountPendingPages(ctx context.Context, sourceID string) (int, error) {
	return a.Repo.CountPendingPages(ctx, sourceID)
}

func TestIngestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	ctx := context.Background()

	// 1. Setup Dependencies
	sourceRepo := source.NewPostgresRepo(s.DB)
	jobRepo := job.NewPostgresRepo(s.DB)
	vectorStore := weaviate.NewStore(s.Weaviate)
	embedder := new(MockEmbedder)
	sourceFetcher := &TestSourceFetcher{Repo: sourceRepo}

	// Ensure Weaviate Schema
	err := vectorStore.EnsureSchema(ctx)
	require.NoError(t, err)

	pageManager := &PageManagerAdapter{Repo: sourceRepo}

	consumer := worker.NewResultConsumer(
		embedder,
		vectorStore,
		sourceRepo, // SourceStatusUpdater
		jobRepo,
		sourceFetcher,
		pageManager, // PageManager
		s.NSQ,       // TaskPublisher (Real NSQ Producer)
	)

	// 2. Setup Data: Create Source & Page
	src := &source.Source{
		Type:        "web",
		URL:         "http://example.com",
		ContentHash: "hash-integration",
		Status:      "in_progress",
		MaxDepth:    1,
		Name:        "Integration Source",
	}
	err = sourceRepo.Save(ctx, src)
	require.NoError(t, err)

	_, err = sourceRepo.BulkCreatePages(ctx, []source.SourcePage{{
		SourceID: src.ID,
		URL:      src.URL,
		Status:   "pending",
		Depth:    0,
	}})
	require.NoError(t, err)

	// 3. Simulate Message Handling (Success)
	payload := map[string]interface{}{
		"source_id": src.ID,
		"url":       src.URL,
		"content":   "# Hello World\nThis is a test page.",
		"title":     "Hello Page",
		"status":    "success",
		"depth":     0,
		"metadata": map[string]interface{}{
			"author": "Test Bot",
		},
	}
	body, _ := json.Marshal(payload)
	msg := &nsq.Message{
		Body:      body,
		ID:        nsq.MessageID{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', 'a', 'b', 'c', 'd', 'e', 'f'},
		Timestamp: time.Now().UnixNano(),
	}

	// Exec HandleMessage
	err = consumer.HandleMessage(msg)
	require.NoError(t, err)

	// 4. Verify Side Effects

	// A. Check Vector Store
	chunks, err := vectorStore.GetChunks(ctx, src.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, chunks)
	assert.Contains(t, chunks[0].Content, "Hello World")
	assert.Equal(t, "Integration Source", chunks[0].SourceName)

	// B. Check Page Status
	pages, err := sourceRepo.GetPages(ctx, src.ID)
	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Equal(t, "completed", pages[0].Status)

	// C. Check Source Status (Should be completed as it was the only page)
	updatedSrc, err := sourceRepo.Get(ctx, src.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updatedSrc.Status)
}

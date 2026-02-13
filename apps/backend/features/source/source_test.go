package source

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/middleware"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/worker"
)

// Reusing MockRepo from handler_test.go is tricky because they are in the same package (source)
// but handler_test.go defines MockRepo and it seems to lack methods.
// To avoid conflict, I'll rename my mocks here.

type TestPublisher struct {
	LastTopic string
	LastBody  []byte
}

func (m *TestPublisher) Publish(topic string, body []byte) error {
	m.LastTopic = topic
	m.LastBody = body
	return nil
}

// Minimal mocks for dependencies
type TestRepo struct{ Repository }

func (m *TestRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) { return false, nil }
func (m *TestRepo) Save(ctx context.Context, src *Source) error                 { return nil }
func (m *TestRepo) Count(ctx context.Context) (int, error)                      { return 0, nil }
func (m *TestRepo) ResetStuckPages(ctx context.Context, timeout time.Duration) (int64, error) {
	return 1, nil
}

func (m *TestRepo) BulkCreatePages(ctx context.Context, pages []SourcePage) ([]string, error) {
	return nil, nil
}

func (m *TestRepo) UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error {
	return nil
}

func (m *TestRepo) GetPages(ctx context.Context, sourceID string) ([]SourcePage, error) {
	return nil, nil
}
func (m *TestRepo) DeletePages(ctx context.Context, sourceID string) error { return nil }
func (m *TestRepo) CountPendingPages(ctx context.Context, sourceID string) (int, error) {
	return 0, nil
}
func (m *TestRepo) Get(ctx context.Context, id string) (*Source, error)       { return nil, nil }
func (m *TestRepo) List(ctx context.Context) ([]Source, error)                { return nil, nil }
func (m *TestRepo) UpdateStatus(ctx context.Context, id, status string) error { return nil }
func (m *TestRepo) UpdateBodyHash(ctx context.Context, id, hash string) error { return nil }
func (m *TestRepo) SoftDelete(ctx context.Context, id string) error           { return nil }

type TestSettings struct{ SettingsService }

func (m *TestSettings) Get(ctx context.Context) (*settings.Settings, error) { return nil, nil }

type TestChunkStore struct{ ChunkStore }

func (m *TestChunkStore) DeleteChunksBySourceID(ctx context.Context, sourceID string) error {
	return nil
}

func (m *TestChunkStore) GetChunks(ctx context.Context, sourceID string, limit, offset int) ([]worker.Chunk, error) {
	return nil, nil
}

func (m *TestChunkStore) CountChunksBySource(ctx context.Context, sourceID string) (int, error) {
	return 0, nil
}

func TestCreate_PropagatesCorrelationID(t *testing.T) {
	pub := &TestPublisher{}
	repo := &TestRepo{}
	chunkStore := &TestChunkStore{}
	settingsSvc := &TestSettings{}

	svc := NewService(repo, pub, chunkStore, settingsSvc)

	ctx := context.Background()
	expectedID := "trace-123"
	ctx = middleware.WithCorrelationID(ctx, expectedID)

	src := &Source{URL: "http://example.com", Type: "web"}
	if err := svc.Create(ctx, src); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(pub.LastBody, &payload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if id, ok := payload["correlation_id"].(string); !ok || id != expectedID {
		t.Errorf("Expected correlation_id %s, got %v", expectedID, payload["correlation_id"])
	}

	if pub.LastTopic != config.TopicIngestWeb {
		t.Errorf("Expected topic %s, got %s", config.TopicIngestWeb, pub.LastTopic)
	}
}

func TestService_Create_InvalidRegex(t *testing.T) {
	repo := &TestRepo{}
	svc := NewService(repo, nil, nil, nil)

	src := &Source{
		URL:        "http://example.com",
		Type:       "web",
		Exclusions: []string{"["}, // Invalid Regex
	}

	err := svc.Create(context.Background(), src)
	if err == nil {
		t.Fatal("Expected error for invalid regex, got nil")
	}
	if err.Error() != "invalid exclusion regex: [" {
		t.Errorf("Expected 'invalid exclusion regex: [', got '%v'", err)
	}
}

func TestCreate_FileSource_PublishesToFileTopic(t *testing.T) {
	pub := &TestPublisher{}
	repo := &TestRepo{}
	svc := NewService(repo, pub, &TestChunkStore{}, &TestSettings{})

	src := &Source{Type: "file", URL: "/tmp/test.pdf"}

	err := svc.Create(context.Background(), src)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if pub.LastTopic != config.TopicIngestFile {
		t.Errorf("Expected topic %s, got %s", config.TopicIngestFile, pub.LastTopic)
	}
}

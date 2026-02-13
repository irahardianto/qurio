package job

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/config"
)

// MockPublisher for Service Test
type MockPublisher struct {
	sleep     time.Duration
	LastTopic string
}

func (m *MockPublisher) Publish(topic string, body []byte) error {
	m.LastTopic = topic
	time.Sleep(m.sleep)
	return nil
}

// MockRepo for Service Test
type MockRepoService struct {
	Repository
}

func (m *MockRepoService) Get(ctx context.Context, id string) (*Job, error) {
	return &Job{ID: id, Payload: []byte("{}")}, nil
}

func (m *MockRepoService) Delete(ctx context.Context, id string) error {
	return nil
}

func TestRetry_Timeout(t *testing.T) {
	repo := &MockRepoService{}
	// Sleep longer than the 5s timeout
	pub := &MockPublisher{sleep: 6 * time.Second}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(repo, pub, logger)

	// We can't wait 6 seconds in a unit test ideally, but to verify the logic we must.
	// Or we could make the timeout configurable in Service, but the plan said "Add 5-second timeout".
	// For this test to be fast, I would ideally inject the timeout duration, but sticking to the plan strictly.
	// Actually, Go test runner has a default timeout of 10m, so 6s is fine for a one-off verification,
	// though not ideal for fast unit tests.
	// Alternatively, I can't easily mock `time.After` without dependency injection.
	// I'll proceed with the 6s sleep for correctness verification as per plan "Verify test passes".

	err := service.Retry(context.Background(), "1")
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
	if err.Error() != "timeout waiting for NSQ publish" {
		t.Errorf("Expected 'timeout waiting for NSQ publish', got '%v'", err)
	}
}

func (m *MockRepoService) Count(ctx context.Context) (int, error) { return 10, nil }
func (m *MockRepoService) List(ctx context.Context) ([]Job, error) {
	return []Job{{ID: "1"}, {ID: "2"}}, nil
}

func TestService_Count(t *testing.T) {
	repo := &MockRepoService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(repo, nil, logger)

	count, err := service.Count(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if count != 10 {
		t.Errorf("Expected count 10, got %d", count)
	}
}

func TestService_List(t *testing.T) {
	repo := &MockRepoService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(repo, nil, logger)

	jobs, err := service.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, jobs, 2)
	assert.Equal(t, "1", jobs[0].ID)
}

func TestService_ResetStuckJobs(t *testing.T) {
	repo := &MockRepoService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewService(repo, nil, logger)

	count, err := service.ResetStuckJobs(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

type MockJobRepoForTopic struct {
	Repository
	Payload []byte
}

func (m *MockJobRepoForTopic) Get(ctx context.Context, id string) (*Job, error) {
	return &Job{ID: id, Payload: m.Payload}, nil
}
func (m *MockJobRepoForTopic) Delete(ctx context.Context, id string) error { return nil }
func (m *MockJobRepoForTopic) List(ctx context.Context) ([]Job, error)     { return nil, nil }
func (m *MockJobRepoForTopic) Count(ctx context.Context) (int, error)      { return 0, nil }
func (m *MockJobRepoForTopic) Save(ctx context.Context, job *Job) error    { return nil }

func TestRetry_TopicSelection(t *testing.T) {
	pub := &MockPublisher{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// We need to inject the specific repo behavior.
	// Since NewService takes Repository interface, we can pass our custom mock.
	customRepo := &MockJobRepoForTopic{
		Payload: []byte(`{"type": "file", "path": "/tmp/test.pdf"}`),
	}

	service := NewService(customRepo, pub, logger)

	err := service.Retry(context.Background(), "1")
	if err != nil {
		t.Fatalf("Retry failed: %v", err)
	}

	if pub.LastTopic != config.TopicIngestFile {
		t.Errorf("Expected topic %s, got %s", config.TopicIngestFile, pub.LastTopic)
	}
}

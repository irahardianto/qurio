package source

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/worker"
)

// --- Mocks ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) BulkCreatePages(ctx context.Context, pages []SourcePage) ([]string, error) {
	args := m.Called(ctx, pages)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRepository) UpdatePageStatus(ctx context.Context, sourceID, url, status, err string) error {
	args := m.Called(ctx, sourceID, url, status, err)
	return args.Error(0)
}

func (m *MockRepository) GetPages(ctx context.Context, sourceID string) ([]SourcePage, error) {
	args := m.Called(ctx, sourceID)
	return args.Get(0).([]SourcePage), args.Error(1)
}

func (m *MockRepository) DeletePages(ctx context.Context, sourceID string) error {
	args := m.Called(ctx, sourceID)
	return args.Error(0)
}

func (m *MockRepository) CountPendingPages(ctx context.Context, sourceID string) (int, error) {
	args := m.Called(ctx, sourceID)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) ResetStuckPages(ctx context.Context, timeout time.Duration) (int64, error) {
	args := m.Called(ctx, timeout)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) Save(ctx context.Context, src *Source) error {
	args := m.Called(ctx, src)
	return args.Error(0)
}

func (m *MockRepository) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	args := m.Called(ctx, hash)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) Get(ctx context.Context, id string) (*Source, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Source), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context) ([]Source, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Source), args.Error(1)
}

func (m *MockRepository) UpdateStatus(ctx context.Context, id, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateBodyHash(ctx context.Context, id, hash string) error {
	args := m.Called(ctx, id, hash)
	return args.Error(0)
}

func (m *MockRepository) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(topic string, body []byte) error {
	args := m.Called(topic, body)
	return args.Error(0)
}

type MockChunkStore struct {
	mock.Mock
}

func (m *MockChunkStore) GetChunks(ctx context.Context, sourceID string) ([]worker.Chunk, error) {
	args := m.Called(ctx, sourceID)
	return args.Get(0).([]worker.Chunk), args.Error(1)
}

func (m *MockChunkStore) DeleteChunksBySourceID(ctx context.Context, sourceID string) error {
	args := m.Called(ctx, sourceID)
	return args.Error(0)
}

type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) Get(ctx context.Context) (*settings.Settings, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settings.Settings), args.Error(1)
}

// --- Tests ---

func TestService_Create_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockPub := new(MockPublisher)
	mockChunk := new(MockChunkStore)
	mockSettings := new(MockSettingsService)

	svc := NewService(mockRepo, mockPub, mockChunk, mockSettings)

	src := &Source{
		ID:  "src-1",
		URL: "https://example.com",
	}

	// 1. Check duplicate
	mockRepo.On("ExistsByHash", mock.Anything, mock.AnythingOfType("string")).Return(false, nil)
	
	// 2. Save
	mockRepo.On("Save", mock.Anything, mock.MatchedBy(func(s *Source) bool {
		return s.Status == "in_progress" && s.Type == "web"
	})).Return(nil)

	// 3. Create Seed Page
	mockRepo.On("BulkCreatePages", mock.Anything, mock.MatchedBy(func(pages []SourcePage) bool {
		return len(pages) == 1 && pages[0].URL == "https://example.com"
	})).Return([]string{"page-1"}, nil)

	// 4. Get Settings
	mockSettings.On("Get", mock.Anything).Return(&settings.Settings{GeminiAPIKey: "key"}, nil)

	// 5. Publish
	mockPub.On("Publish", "ingest.task", mock.Anything).Return(nil)

	err := svc.Create(context.Background(), src)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockPub.AssertExpectations(t)
}

func TestService_Create_Duplicate(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo, nil, nil, nil)

	src := &Source{URL: "https://example.com"}

	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(true, nil)

	err := svc.Create(context.Background(), src)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Duplicate")
}

func TestService_Delete(t *testing.T) {
	mockRepo := new(MockRepository)
	mockChunk := new(MockChunkStore)
	svc := NewService(mockRepo, nil, mockChunk, nil)

	id := "src-1"

	// 1. Delete Chunks
	mockChunk.On("DeleteChunksBySourceID", mock.Anything, id).Return(nil)

	// 2. Soft Delete
	mockRepo.On("SoftDelete", mock.Anything, id).Return(nil)

	err := svc.Delete(context.Background(), id)
	assert.NoError(t, err)
	mockChunk.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestService_ReSync(t *testing.T) {
	mockRepo := new(MockRepository)
	mockPub := new(MockPublisher)
	mockSettings := new(MockSettingsService)
	svc := NewService(mockRepo, mockPub, nil, mockSettings)

	id := "src-1"
	src := &Source{ID: id, URL: "https://example.com", Type: "web"}

	// 1. Get Source
	mockRepo.On("Get", mock.Anything, id).Return(src, nil)

	// 2. Update Status
	mockRepo.On("UpdateStatus", mock.Anything, id, "in_progress").Return(nil)

	// 3. Delete Pages
	mockRepo.On("DeletePages", mock.Anything, id).Return(nil)

	// 4. Create Seed Page
	mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{"p1"}, nil)

	// 5. Settings
	mockSettings.On("Get", mock.Anything).Return(nil, errors.New("no settings")) // Fallback to empty key

	// 6. Publish
	mockPub.On("Publish", "ingest.task", mock.MatchedBy(func(body []byte) bool {
		var m map[string]interface{}
		json.Unmarshal(body, &m)
		return m["resync"] == true
	})).Return(nil)

	err := svc.ReSync(context.Background(), id)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockPub.AssertExpectations(t)
}

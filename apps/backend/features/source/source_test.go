package source_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"crypto/sha256"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/worker"
	"qurio/apps/backend/internal/settings"
)

type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) Save(ctx context.Context, src *source.Source) error {
	args := m.Called(ctx, src)
	return args.Error(0)
}

func (m *MockRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	args := m.Called(ctx, hash)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepo) List(ctx context.Context) ([]source.Source, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]source.Source), args.Error(1)
}

func (m *MockRepo) UpdateStatus(ctx context.Context, id, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockRepo) Get(ctx context.Context, id string) (*source.Source, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*source.Source), args.Error(1)
}

func (m *MockRepo) UpdateBodyHash(ctx context.Context, id, hash string) error {
	args := m.Called(ctx, id, hash)
	return args.Error(0)
}

func (m *MockRepo) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
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

func TestCreateSource(t *testing.T) {
	repo := new(MockRepo)
	pub := new(MockPublisher)
	chunkStore := new(MockChunkStore)
	settingsMock := new(MockSettingsService)
	svc := source.NewService(repo, pub, chunkStore, settingsMock)
	
	src := &source.Source{URL: "https://example.com"}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(src.URL)))

	// Expect ExistsByHash -> false
	repo.On("ExistsByHash", mock.Anything, hash).Return(false, nil)
	
	// Expect Save -> success
	repo.On("Save", mock.Anything, mock.MatchedBy(func(s *source.Source) bool {
		return s.URL == src.URL && s.ContentHash == hash
	})).Return(nil)
	
	// Expect Settings -> success
	settingsMock.On("Get", mock.Anything).Return(&settings.Settings{GeminiAPIKey: "test-key"}, nil)

	// Expect Publish -> success
	pub.On("Publish", "ingest.task", mock.MatchedBy(func(body []byte) bool {
		var p map[string]interface{}
		json.Unmarshal(body, &p)
		return p["gemini_api_key"] == "test-key"
	})).Return(nil)
	
	err := svc.Create(context.Background(), src)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
	pub.AssertExpectations(t)
	settingsMock.AssertExpectations(t)
}

func TestCreateSource_Duplicate(t *testing.T) {
	repo := new(MockRepo)
	pub := new(MockPublisher)
	chunkStore := new(MockChunkStore)
	settingsMock := new(MockSettingsService)
	svc := source.NewService(repo, pub, chunkStore, settingsMock)
	
	src := &source.Source{URL: "https://duplicate.com"}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(src.URL)))

	// Expect ExistsByHash -> true
	repo.On("ExistsByHash", mock.Anything, hash).Return(true, nil)
	
	err := svc.Create(context.Background(), src)
	
	assert.Error(t, err)
	assert.Equal(t, "Duplicate detected", err.Error())
	
	// Save and Publish should NOT be called
	repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	pub.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

func TestDeleteSource(t *testing.T) {
	repo := new(MockRepo)
	pub := new(MockPublisher)
	chunkStore := new(MockChunkStore)
	settingsMock := new(MockSettingsService)
	svc := source.NewService(repo, pub, chunkStore, settingsMock)

	id := "some-id"
	repo.On("SoftDelete", mock.Anything, id).Return(nil)

	err := svc.Delete(context.Background(), id)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestReSyncSource(t *testing.T) {
	repo := new(MockRepo)
	pub := new(MockPublisher)
	chunkStore := new(MockChunkStore)
	settingsMock := new(MockSettingsService)
	svc := source.NewService(repo, pub, chunkStore, settingsMock)

	id := "some-id"
	src := &source.Source{ID: id, URL: "http://example.com"}

	repo.On("Get", mock.Anything, id).Return(src, nil)
	repo.On("UpdateStatus", mock.Anything, id, "in_progress").Return(nil)
	settingsMock.On("Get", mock.Anything).Return(&settings.Settings{GeminiAPIKey: "test-key"}, nil)
	
	pub.On("Publish", "ingest.task", mock.MatchedBy(func(body []byte) bool {
		var p map[string]interface{}
		json.Unmarshal(body, &p)
		return p["resync"] == true && p["gemini_api_key"] == "test-key"
	})).Return(nil)

	err := svc.ReSync(context.Background(), id)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreateSource_WithConfig(t *testing.T) {
	repo := new(MockRepo)
	pub := new(MockPublisher)
	chunkStore := new(MockChunkStore)
	settingsMock := new(MockSettingsService)
	svc := source.NewService(repo, pub, chunkStore, settingsMock)
	
	src := &source.Source{
		URL:        "https://example.com",
		MaxDepth:   2,
		Exclusions: []string{"/admin", "/login"},
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(src.URL)))

	repo.On("ExistsByHash", mock.Anything, hash).Return(false, nil)
	repo.On("Save", mock.Anything, mock.Anything).Return(nil)
	settingsMock.On("Get", mock.Anything).Return(&settings.Settings{}, nil)
	
	pub.On("Publish", "ingest.task", mock.MatchedBy(func(body []byte) bool {
		var p map[string]interface{}
		json.Unmarshal(body, &p)
		
		maxDepth, ok := p["max_depth"].(float64)
		if !ok || maxDepth != 2 {
			return false
		}
		
		exclusions, ok := p["exclusions"].([]interface{})
		if !ok || len(exclusions) != 2 {
			return false
		}
		
		return p["url"] == src.URL
	})).Return(nil)
	
	err := svc.Create(context.Background(), src)
	assert.NoError(t, err)
	pub.AssertExpectations(t)
}
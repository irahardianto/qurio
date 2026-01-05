package source_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/worker"
)

// MockRepo implements source.Repository
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) Save(ctx context.Context, src *source.Source) error {
	args := m.Called(ctx, src)
	return args.Error(0)
}
func (m *MockRepo) List(ctx context.Context) ([]source.Source, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]source.Source), args.Error(1)
}
func (m *MockRepo) Get(ctx context.Context, id string) (*source.Source, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*source.Source), args.Error(1)
}
func (m *MockRepo) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	args := m.Called(ctx, hash)
	return args.Bool(0), args.Error(1)
}
func (m *MockRepo) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockRepo) UpdateStatus(ctx context.Context, id, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}
func (m *MockRepo) UpdateBodyHash(ctx context.Context, id, hash string) error {
	args := m.Called(ctx, id, hash)
	return args.Error(0)
}
func (m *MockRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}
func (m *MockRepo) BulkCreatePages(ctx context.Context, pages []source.SourcePage) ([]string, error) {
	args := m.Called(ctx, pages)
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockRepo) UpdatePageStatus(ctx context.Context, sourceID, url, status, errStr string) error {
	args := m.Called(ctx, sourceID, url, status, errStr)
	return args.Error(0)
}
func (m *MockRepo) GetPages(ctx context.Context, sourceID string) ([]source.SourcePage, error) {
	args := m.Called(ctx, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]source.SourcePage), args.Error(1)
}
func (m *MockRepo) DeletePages(ctx context.Context, sourceID string) error {
	args := m.Called(ctx, sourceID)
	return args.Error(0)
}
func (m *MockRepo) CountPendingPages(ctx context.Context, sourceID string) (int, error) {
	args := m.Called(ctx, sourceID)
	return args.Int(0), args.Error(1)
}
func (m *MockRepo) ResetStuckPages(ctx context.Context, timeout time.Duration) (int64, error) {
	args := m.Called(ctx, timeout)
	return args.Get(0).(int64), args.Error(1)
}

// MockChunkStore
type MockChunkStore struct {
	mock.Mock
}

func (m *MockChunkStore) GetChunks(ctx context.Context, sourceID string) ([]worker.Chunk, error) {
	args := m.Called(ctx, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]worker.Chunk), args.Error(1)
}

func (m *MockChunkStore) DeleteChunksBySourceID(ctx context.Context, sourceID string) error {
	args := m.Called(ctx, sourceID)
	return args.Error(0)
}

// MockSettingsService
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

// MockPublisher
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(topic string, body []byte) error {
	args := m.Called(topic, body)
	return args.Error(0)
}

func TestHandler_Create(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockPub := new(MockPublisher)
		mockSettings := new(MockSettingsService)
		svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
		handler := source.NewHandler(svc)

		mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, nil)
		mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{}, nil)
		mockSettings.On("Get", mock.Anything).Return(&settings.Settings{}, nil)
		mockPub.On("Publish", "ingest.task", mock.Anything).Return(nil)

		reqBody := `{"type": "web", "url": "http://example.com", "max_depth": 1}`
		req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		handler.Create(w, req)

		assert.Equal(t, http.StatusCreated, w.Result().StatusCode)
	})

	t.Run("Duplicate", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockPub := new(MockPublisher)
		mockSettings := new(MockSettingsService)
		svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
		handler := source.NewHandler(svc)

		mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(true, nil)

		reqBody := `{"type": "web", "url": "http://dup.com"}`
		req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		handler.Create(w, req)

		assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
	})
}

func TestHandler_ReSync(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
	handler := source.NewHandler(svc)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Get", mock.Anything, "1").Return(&source.Source{ID: "1", Type: "web", URL: "http://example.com"}, nil)
		mockRepo.On("UpdateStatus", mock.Anything, "1", "in_progress").Return(nil)
		mockRepo.On("DeletePages", mock.Anything, "1").Return(nil)
		mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{}, nil)
		mockSettings.On("Get", mock.Anything).Return(&settings.Settings{}, nil)
		mockPub.On("Publish", "ingest.task", mock.Anything).Return(nil)

		req := httptest.NewRequest("POST", "/sources/1/resync", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.ReSync(w, req)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	})
}

func TestHandler_List(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil) // nil nsq, nil vector, nil settings
	handler := source.NewHandler(svc)

	mockRepo.On("List", mock.Anything).Return([]source.Source{{ID: "1"}}, nil)

	req := httptest.NewRequest("GET", "/sources", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	mockRepo.AssertExpectations(t)
}

func TestHandler_Delete(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc)

	mockRepo.On("SoftDelete", mock.Anything, "1").Return(nil)
	mockChunkStore.On("DeleteChunksBySourceID", mock.Anything, "1").Return(nil)
	
	req := httptest.NewRequest("DELETE", "/sources/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Delete(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_Get(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc)

	mockRepo.On("Get", mock.Anything, "1").Return(&source.Source{ID: "1"}, nil)
	mockChunkStore.On("GetChunks", mock.Anything, "1").Return([]worker.Chunk{}, nil)

	req := httptest.NewRequest("GET", "/sources/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_GetPages(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc)

	mockRepo.On("GetPages", mock.Anything, "1").Return([]source.SourcePage{}, nil)

	req := httptest.NewRequest("GET", "/sources/1/pages", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.GetPages(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

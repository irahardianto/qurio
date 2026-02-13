package source_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/settings"
	"qurio/apps/backend/internal/worker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockChunkStore) GetChunks(ctx context.Context, sourceID string, limit, offset int) ([]worker.Chunk, error) {
	args := m.Called(ctx, sourceID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]worker.Chunk), args.Error(1)
}

func (m *MockChunkStore) DeleteChunksBySourceID(ctx context.Context, sourceID string) error {
	args := m.Called(ctx, sourceID)
	return args.Error(0)
}

func (m *MockChunkStore) CountChunksBySource(ctx context.Context, sourceID string) (int, error) {
	args := m.Called(ctx, sourceID)
	return args.Int(0), args.Error(1)
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
		handler := source.NewHandler(svc, t.TempDir(), 50)

		mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, nil)
		mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{}, nil)
		mockSettings.On("Get", mock.Anything).Return(&settings.Settings{}, nil)
		mockPub.On("Publish", config.TopicIngestWeb, mock.Anything).Return(nil)

		reqBody := `{"type": "web", "url": "http://example.com", "max_depth": 1, "name": "Test Web"}`
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
		handler := source.NewHandler(svc, t.TempDir(), 50)

		mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(true, nil)

		reqBody := `{"type": "web", "url": "http://dup.com", "name": "Duplicate Web"}`
		req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		handler.Create(w, req)

		assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
	})

	t.Run("InvalidInput", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockPub := new(MockPublisher)
		mockSettings := new(MockSettingsService)
		svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
		handler := source.NewHandler(svc, t.TempDir(), 50)

		// Missing URL
		reqBody := `{"type": "web", "url": ""}`
		req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		handler.Create(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}

func TestHandler_Upload(t *testing.T) {
	// Mock Service logic for Upload is complicated due to file I/O.
	// We'll focus on testing the Handler validation logic here.
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	t.Run("Invalid File Type", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.exe")
		part.Write([]byte("binary"))
		writer.WriteField("name", "Test Name")
		writer.Close()

		req := httptest.NewRequest("POST", "/sources/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler.Upload(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)

		var resp map[string]interface{}
		json.NewDecoder(w.Result().Body).Decode(&resp)
		errObj := resp["error"].(map[string]interface{})
		assert.Equal(t, "Unsupported file type", errObj["message"])
	})
}

func TestHandler_ReSync(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Get", mock.Anything, "1").Return(&source.Source{ID: "1", Type: "web", URL: "http://example.com"}, nil)
		mockRepo.On("UpdateStatus", mock.Anything, "1", "in_progress").Return(nil)
		mockRepo.On("DeletePages", mock.Anything, "1").Return(nil)
		mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{}, nil)
		mockSettings.On("Get", mock.Anything).Return(&settings.Settings{}, nil)
		mockPub.On("Publish", config.TopicIngestWeb, mock.Anything).Return(nil)

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
	handler := source.NewHandler(svc, t.TempDir(), 50)

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
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("SoftDelete", mock.Anything, "1").Return(nil)
	mockChunkStore.On("DeleteChunksBySourceID", mock.Anything, "1").Return(nil)

	req := httptest.NewRequest("DELETE", "/sources/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Delete(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_Delete_NotFound(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockChunkStore.On("DeleteChunksBySourceID", mock.Anything, "99").Return(nil)
	mockRepo.On("SoftDelete", mock.Anything, "99").Return(sql.ErrNoRows)

	req := httptest.NewRequest("DELETE", "/sources/99", nil)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()

	handler.Delete(w, req)
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestHandler_Get(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("Get", mock.Anything, "1").Return(&source.Source{ID: "1"}, nil)
	mockChunkStore.On("CountChunksBySource", mock.Anything, "1").Return(10, nil)
	mockChunkStore.On("GetChunks", mock.Anything, "1", 100, 0).Return([]worker.Chunk{}, nil)

	req := httptest.NewRequest("GET", "/sources/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_Get_Pagination(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("Get", mock.Anything, "1").Return(&source.Source{ID: "1"}, nil)
	mockChunkStore.On("CountChunksBySource", mock.Anything, "1").Return(200, nil)
	mockChunkStore.On("GetChunks", mock.Anything, "1", 20, 10).Return([]worker.Chunk{}, nil)

	req := httptest.NewRequest("GET", "/sources/1?limit=20&offset=10", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_Get_ExcludeChunks(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("Get", mock.Anything, "1").Return(&source.Source{ID: "1"}, nil)
	mockChunkStore.On("CountChunksBySource", mock.Anything, "1").Return(200, nil)
	// GetChunks shouldn't be called

	req := httptest.NewRequest("GET", "/sources/1?exclude_chunks=true", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	mockChunkStore.AssertNotCalled(t, "GetChunks")
}

func TestHandler_Get_NotFound(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("Get", mock.Anything, "99").Return(nil, sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/sources/99", nil)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()

	handler.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestHandler_GetPages(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, nil, mockChunkStore, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("GetPages", mock.Anything, "1").Return([]source.SourcePage{}, nil)

	req := httptest.NewRequest("GET", "/sources/1/pages", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.GetPages(w, req)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_Upload_DefaultDirectory(t *testing.T) {
	uploadDir := t.TempDir()

	// Mock Dependencies
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	// Service needs repo and publisher (mocked)
	svc := source.NewService(mockRepo, mockPub, nil, nil)
	handler := source.NewHandler(svc, uploadDir, 50)

	// Mock Expectations
	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, nil)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
	// Publisher is called in Service.Upload ("ingest.task")
	mockPub.On("Publish", config.TopicIngestFile, mock.Anything).Return(nil)

	// Prepare Request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "default_dir.txt")
	part.Write([]byte("content"))
	writer.WriteField("name", "Test File")
	writer.Close()

	req := httptest.NewRequest("POST", "/sources/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Execute
	handler.Upload(w, req)

	// Assertions
	if !assert.Equal(t, http.StatusCreated, w.Result().StatusCode) {
		t.Logf("Response: %s", w.Body.String())
	}

	// Verify upload directory contains the file
	entries, err := os.ReadDir(uploadDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, entries)
}

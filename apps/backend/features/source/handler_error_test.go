package source_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_Create_InvalidJSON(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	req := httptest.NewRequest("POST", "/sources", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	handler.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandler_Create_InternalServiceError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	// ExistsByHash fails
	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, errors.New("db error"))

	reqBody := `{"type": "web", "url": "http://example.com", "name": "Test"}`
	req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	handler.Create(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandler_Delete_InternalError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	svc := source.NewService(mockRepo, nil, mockChunkStore, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	// Chunk deletion fails with a non-sql.ErrNoRows error
	mockChunkStore.On("DeleteChunksBySourceID", mock.Anything, "1").Return(errors.New("weaviate timeout"))

	req := httptest.NewRequest("DELETE", "/sources/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Delete(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandler_Upload_DuplicateHash(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	svc := source.NewService(mockRepo, mockPub, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(true, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.pdf")
	part.Write([]byte("content"))
	writer.WriteField("name", "Test File")
	writer.Close()

	req := httptest.NewRequest("POST", "/sources/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.Upload(w, req)
	assert.Equal(t, http.StatusConflict, w.Result().StatusCode)
}

func TestHandler_Upload_SaveError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	svc := source.NewService(mockRepo, mockPub, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, nil)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("content"))
	writer.WriteField("name", "Test File")
	writer.Close()

	req := httptest.NewRequest("POST", "/sources/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.Upload(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandler_Create_MissingURL(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	reqBody := `{"type": "web", "name": "No URL"}`
	req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	handler.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandler_Upload_FileTooLarge(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	// maxUploadSizeMB = 1 (1 MB)
	handler := source.NewHandler(svc, t.TempDir(), 1)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "large.txt")
	// Write more than 1MB of data
	largeData := make([]byte, 2*1024*1024) // 2 MB
	part.Write(largeData)
	writer.WriteField("name", "Large File")
	writer.Close()

	req := httptest.NewRequest("POST", "/sources/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.Upload(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandler_Get_InternalError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockChunkStore := new(MockChunkStore)
	svc := source.NewService(mockRepo, nil, mockChunkStore, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("Get", mock.Anything, "1").Return(nil, errors.New("db connection failed"))

	req := httptest.NewRequest("GET", "/sources/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.Get(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandler_Create_SaveError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, nil)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db write error"))

	reqBody := `{"type": "web", "url": "http://example.com", "name": "Test"}`
	req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	handler.Create(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandler_Create_PublishError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, nil)
	mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{"p1"}, nil)
	mockSettings.On("Get", mock.Anything).Return(&settings.Settings{}, nil)
	mockPub.On("Publish", config.TopicIngestWeb, mock.Anything).Return(errors.New("nsq error"))

	reqBody := `{"type": "web", "url": "http://example.com", "name": "Test"}`
	req := httptest.NewRequest("POST", "/sources", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	handler.Create(w, req)
	// Service logs but swallows publish errors — Create still succeeds
	assert.Equal(t, http.StatusCreated, w.Result().StatusCode)
}

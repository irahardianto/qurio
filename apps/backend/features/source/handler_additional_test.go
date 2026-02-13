package source_test

import (
	"bytes"
	"database/sql"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_ReSync_ErrorPaths(t *testing.T) {
	t.Run("NotFound", func(t *testing.T) {
		mockRepo := new(MockRepo)
		svc := source.NewService(mockRepo, nil, nil, nil) // minimal deps
		handler := source.NewHandler(svc, t.TempDir(), 50)

		mockRepo.On("Get", mock.Anything, "99").Return(nil, sql.ErrNoRows)

		req := httptest.NewRequest("POST", "/sources/99/resync", nil)
		req.SetPathValue("id", "99")
		w := httptest.NewRecorder()

		handler.ReSync(w, req)

		assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	})

	t.Run("UpdateStatusError", func(t *testing.T) {
		mockRepo := new(MockRepo)
		svc := source.NewService(mockRepo, nil, nil, nil)
		handler := source.NewHandler(svc, t.TempDir(), 50)

		src := &source.Source{ID: "1", Type: "web", URL: "http://example.com"}
		mockRepo.On("Get", mock.Anything, "1").Return(src, nil)
		mockRepo.On("UpdateStatus", mock.Anything, "1", "in_progress").Return(errors.New("db error"))

		req := httptest.NewRequest("POST", "/sources/1/resync", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.ReSync(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("DeletePagesError", func(t *testing.T) {
		mockRepo := new(MockRepo)
		svc := source.NewService(mockRepo, nil, nil, nil)
		handler := source.NewHandler(svc, t.TempDir(), 50)

		src := &source.Source{ID: "1", Type: "web", URL: "http://example.com"}
		mockRepo.On("Get", mock.Anything, "1").Return(src, nil)
		mockRepo.On("UpdateStatus", mock.Anything, "1", "in_progress").Return(nil)
		mockRepo.On("DeletePages", mock.Anything, "1").Return(errors.New("cleanup error"))

		req := httptest.NewRequest("POST", "/sources/1/resync", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.ReSync(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("BulkCreatePagesError", func(t *testing.T) {
		mockRepo := new(MockRepo)
		svc := source.NewService(mockRepo, nil, nil, nil)
		handler := source.NewHandler(svc, t.TempDir(), 50)

		src := &source.Source{ID: "1", Type: "web", URL: "http://example.com"}
		mockRepo.On("Get", mock.Anything, "1").Return(src, nil)
		mockRepo.On("UpdateStatus", mock.Anything, "1", "in_progress").Return(nil)
		mockRepo.On("DeletePages", mock.Anything, "1").Return(nil)
		// Return empty slice instead of nil to avoid panic in mock type assertion
		mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{}, errors.New("seed creation error"))

		req := httptest.NewRequest("POST", "/sources/1/resync", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.ReSync(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})

	t.Run("PublishError", func(t *testing.T) {
		mockRepo := new(MockRepo)
		mockPub := new(MockPublisher)
		mockSettings := new(MockSettingsService)
		svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
		handler := source.NewHandler(svc, t.TempDir(), 50)

		src := &source.Source{ID: "1", Type: "web", URL: "http://example.com"}
		mockRepo.On("Get", mock.Anything, "1").Return(src, nil)
		mockRepo.On("UpdateStatus", mock.Anything, "1", "in_progress").Return(nil)
		mockRepo.On("DeletePages", mock.Anything, "1").Return(nil)
		mockRepo.On("BulkCreatePages", mock.Anything, mock.Anything).Return([]string{}, nil)
		mockSettings.On("Get", mock.Anything).Return(&settings.Settings{}, nil)
		mockPub.On("Publish", config.TopicIngestWeb, mock.Anything).Return(errors.New("nsq error"))

		req := httptest.NewRequest("POST", "/sources/1/resync", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.ReSync(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	})
}

func TestHandler_List_Empty(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("List", mock.Anything).Return([]source.Source{}, nil)

	req := httptest.NewRequest("GET", "/sources", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_List_ServiceError(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("List", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/sources", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandler_GetPages_ServiceError(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("GetPages", mock.Anything, "1").Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/sources/1/pages", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.GetPages(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandler_Upload_MissingName(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("content"))
	// Missing "name" field
	writer.Close()

	req := httptest.NewRequest("POST", "/sources/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.Upload(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandler_Upload_ServiceError(t *testing.T) {
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	svc := source.NewService(mockRepo, mockPub, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	mockRepo.On("ExistsByHash", mock.Anything, mock.Anything).Return(false, errors.New("db error"))

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

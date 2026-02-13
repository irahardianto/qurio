package source_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/features/source"
)

func TestUpload_MissingName(t *testing.T) {
	// Setup Service with Mocks
	mockRepo := new(MockRepo)
	svc := source.NewService(mockRepo, nil, nil, nil)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	// Multipart Request without name
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/sources/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.Upload(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

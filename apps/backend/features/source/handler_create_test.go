package source_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/features/source"
)

func TestCreateSource_MissingName(t *testing.T) {
	// Setup Mock Service (Using shared mocks from handler_test.go)
	mockRepo := new(MockRepo)
	mockPub := new(MockPublisher)
	mockSettings := new(MockSettingsService)
	svc := source.NewService(mockRepo, mockPub, nil, mockSettings)
	handler := source.NewHandler(svc, t.TempDir(), 50)

	// Missing Name
	body := []byte(`{"url":"https://example.com","type":"web"}`)
	req := httptest.NewRequest("POST", "/sources", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

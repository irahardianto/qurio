package source_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qurio/apps/backend/features/source"
	"qurio/apps/backend/internal/testutils"
)

func TestHandler_Upload_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	s := testutils.NewIntegrationSuite(t)
	s.Setup()
	defer s.Teardown()

	// Setup Config for Upload Dir
	tmpDir := t.TempDir()
	os.Setenv("QURIO_UPLOAD_DIR", tmpDir)
	defer os.Unsetenv("QURIO_UPLOAD_DIR")

	// Dependencies
	repo := source.NewPostgresRepo(s.DB)
	service := source.NewService(repo, s.NSQ, nil, nil) // ChunkStore and Settings not needed for Upload
	h := source.NewHandler(service)

	// Prepare File Upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test_doc.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("dummy pdf content"))
	require.NoError(t, err)
	writer.Close()

	// Request
	req := httptest.NewRequest("POST", "/sources/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Execute
	h.Upload(w, req)

	// Assert Response
	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var respBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	data := respBody["data"].(map[string]interface{})
	assert.Equal(t, "file", data["type"])
	assert.Equal(t, "in_progress", data["status"])

	// Verify File on Disk
	// The path in DB (URL field) is the full path
	savedPath := data["url"].(string)
	assert.Contains(t, savedPath, tmpDir)
	assert.FileExists(t, savedPath)
	
	content, err := os.ReadFile(savedPath)
	require.NoError(t, err)
	assert.Equal(t, "dummy pdf content", string(content))
}

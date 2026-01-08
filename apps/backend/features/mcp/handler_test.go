package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestHandler_HandleMessage_MissingSessionID(t *testing.T) {
	handler := NewHandler(nil, nil) // Dependencies not needed for this check

	req := httptest.NewRequest(http.MethodPost, "/mcp/messages", nil)
	// Missing sessionId query param
	
	rec := httptest.NewRecorder()
	handler.HandleMessage(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	
	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	
	assert.Equal(t, "error", resp["status"])
	if errMap, ok := resp["error"].(map[string]interface{}); ok {
		assert.Equal(t, "VALIDATION_ERROR", errMap["code"])
	} else {
		t.Error("Expected error object in response")
	}
}

func TestHandler_HandleMessage_SessionNotFound(t *testing.T) {
	handler := NewHandler(nil, nil)
	
	req := httptest.NewRequest(http.MethodPost, "/mcp/messages?sessionId=unknown-session", nil)
	rec := httptest.NewRecorder()
	
	handler.HandleMessage(rec, req)
	
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_HandleMessage_InvalidJSON(t *testing.T) {
	handler := NewHandler(nil, nil)
	
	// Manually inject a session into the map since it's a private field and we are in the same package
	
	sessionID := "test-session"
	handler.sessions[sessionID] = make(chan string, 1)
	
	req := httptest.NewRequest(http.MethodPost, "/mcp/messages?sessionId="+sessionID, bytes.NewBufferString("{invalid-json"))
	rec := httptest.NewRecorder()
	
	handler.HandleMessage(rec, req)
	
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if errMap, ok := resp["error"].(map[string]interface{}); ok {
		assert.Equal(t, "INVALID_JSON", errMap["code"])
	}
}

func TestHandler_HandleMessage_Success(t *testing.T) {
	// For success, we need to verify it returns 202 Accepted.
	handler := NewHandler(nil, nil)
	sessionID := "test-session"
	handler.sessions[sessionID] = make(chan string, 1)
	
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method": "ping", 
		"id": 1,
	}
	body, _ := json.Marshal(payload)
	
	req := httptest.NewRequest(http.MethodPost, "/mcp/messages?sessionId="+sessionID, bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	
	handler.HandleMessage(rec, req)
	
	assert.Equal(t, http.StatusAccepted, rec.Code)
}

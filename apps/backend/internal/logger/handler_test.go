package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"qurio/apps/backend/internal/middleware"
	"testing"
)

func TestContextHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, nil)
	h := NewContextHandler(jsonHandler)
	logger := slog.New(h)

	ctx := context.Background()
	ctx = middleware.WithCorrelationID(ctx, "test-correlation-id")

	logger.InfoContext(ctx, "test message")

	var logMap map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logMap); err != nil {
		t.Fatalf("failed to unmarshal log: %v", err)
	}

	if logMap["correlation_id"] != "test-correlation-id" {
		t.Errorf("expected correlation_id 'test-correlation-id', got %v", logMap["correlation_id"])
	}
}

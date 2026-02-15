package retrieval

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"
)

func TestQueryLogger_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	logger := NewQueryLogger(&buf)

	concurrency := 50
	iterations := 100
	var wg sync.WaitGroup

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				logger.Log(QueryLogEntry{
					Query:    "test",
					Duration: time.Millisecond,
				})
			}
		}()
	}
	wg.Wait()

	// Verify output is valid JSON stream
	decoder := json.NewDecoder(&buf)
	count := 0
	for decoder.More() {
		var entry QueryLogEntry
		err := decoder.Decode(&entry)
		if err != nil {
			t.Fatalf("Failed to decode entry %d: %v", count, err)
		}
		count++
	}

	expected := concurrency * iterations
	if count != expected {
		t.Errorf("Expected %d entries, got %d", expected, count)
	}
}

func TestNewFileQueryLogger_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/logs/query.log"

	logger, err := NewFileQueryLogger(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify logger works
	logger.Log(QueryLogEntry{
		Query:    "test query",
		Duration: 100 * time.Millisecond,
	})

	// Verify file was created and written
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var entry QueryLogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("failed to decode log entry: %v", err)
	}
	if entry.Query != "test query" {
		t.Errorf("expected query 'test query', got '%s'", entry.Query)
	}
	if entry.LatencyMs != 100 {
		t.Errorf("expected latency 100ms, got %dms", entry.LatencyMs)
	}
}

func TestNewFileQueryLogger_InvalidPath(t *testing.T) {
	// Use a path that cannot be created
	_, err := NewFileQueryLogger("/dev/null/impossible/path/query.log")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

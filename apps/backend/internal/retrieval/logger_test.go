package retrieval

import (
	"bytes"
	"encoding/json"
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

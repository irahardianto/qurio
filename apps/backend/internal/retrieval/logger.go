package retrieval

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type QueryLogEntry struct {
	Timestamp     time.Time     `json:"timestamp"`
	Query         string        `json:"query"`
	NumResults    int           `json:"num_results"`
	Duration      time.Duration `json:"duration_ns"`
	LatencyMs     int64         `json:"latency_ms"`
	CorrelationID string        `json:"correlation_id"`
}

type QueryLogger struct {
	writer io.Writer
	mu     sync.Mutex
}

func NewQueryLogger(w io.Writer) *QueryLogger {
	return &QueryLogger{writer: w}
}

func NewFileQueryLogger(path string) (*QueryLogger, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}

	cleanPath := filepath.Clean(path)
	f, err := os.OpenFile(cleanPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- path is from application config, not user input
	if err != nil {
		return nil, err
	}
	mw := io.MultiWriter(os.Stdout, f)
	return NewQueryLogger(mw), nil
}

func (l *QueryLogger) Log(entry QueryLogEntry) {
	entry.Timestamp = time.Now()
	entry.LatencyMs = entry.Duration.Milliseconds()

	l.mu.Lock()
	defer l.mu.Unlock()
	if err := json.NewEncoder(l.writer).Encode(entry); err != nil {
		slog.Error("failed to write query log entry", "error", err)
	}
}

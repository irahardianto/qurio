### Task 1: Infrastructure - Context-Aware Logger

**Files:**
- Create: `apps/backend/internal/logger/handler.go`
- Modify: `apps/backend/main.go:34-36` (Init logger)
- Test: `apps/backend/internal/logger/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `slog.InfoContext(ctx, "msg")` automatically includes `correlation_id` if present in context.
  2. Logger output defaults to JSON format.
  3. `apps/backend/main.go` initializes this custom handler.

- **Functional Requirements**
  1. Implement `slog.Handler` interface (Decorator pattern).
  2. Extract `correlation_id` from context using `middleware.CorrelationIDKey`.

- **Non-Functional Requirements**
  - Thread-safe.
  - Zero allocation preference (if possible, but standard slog is fine).

- **Test Coverage**
  - [Unit] `TestContextHandler_Handle` - verify correlation_id appears in output JSON.
  - [Integration] None (Unit covers it).

**Step 1: Write failing test**
```go
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
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/logger/... -v`
Expected: FAIL (Compilation error: NewContextHandler undefined)

**Step 3: Write minimal implementation**
```go
package logger

import (
	"context"
	"log/slog"
	"qurio/apps/backend/internal/middleware"
)

type ContextHandler struct {
	slog.Handler
}

func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: h}
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(middleware.CorrelationIDKey).(string); ok {
		r.AddAttrs(slog.String("correlation_id", id))
	}
	return h.Handler.Handle(ctx, r)
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/logger/... -v`
Expected: PASS

### Task 2: Observability - Adapter Logging

**Files:**
- Modify: `apps/backend/internal/adapter/gemini/embedder.go`
- Modify: `apps/backend/internal/adapter/weaviate/store.go`
- Test: `apps/backend/internal/adapter/gemini/embedder_test.go` (if exists, else create mock test)

**Requirements:**
- **Acceptance Criteria**
  1. `Embedder.Embed` logs DEBUG on start, ERROR on fail.
  2. `Store.StoreChunk` logs DEBUG on start, ERROR on fail.
  3. `Store.Search` logs DEBUG on start (with query/alpha), ERROR on fail.

- **Functional Requirements**
  1. Use `slog.DebugContext` and `slog.ErrorContext`.
  2. Include `correlation_id` (handled by Task 1 logger).

- **Non-Functional Requirements**
  - Minimal performance impact (DEBUG logs should be fast to skip if disabled).

- **Test Coverage**
  - Manual verification via logs or mock logger injection (since we use global `slog`, unit testing log output is tricky without dependency injection of logger, but we can rely on integration verification).

**Step 1: Write failing test (Verification Script)**
*Since these are side-effects (logs), we'll verify by running the updated code or adding a unit test that captures stdout. Given the simple nature, we will modify the code directly and verify via compilation and manual check, or add a test that swaps `slog.Default()`.*

**Step 3: Write minimal implementation (Embedder)**
```go
// apps/backend/internal/adapter/gemini/embedder.go
// Add import "log/slog"

func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
    slog.DebugContext(ctx, "embedding content", "model", e.model, "length", len(text))
	em := e.client.EmbeddingModel(e.model)
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
        slog.ErrorContext(ctx, "embedding failed", "error", err)
		return nil, err
	}
    // ...
```

**Step 4: Verify test passes**
Run: `go build ./apps/backend/internal/adapter/gemini/...`

### Task 3: Refactor - Link Discovery Pure Function

**Files:**
- Create: `apps/backend/internal/worker/link_discovery.go`
- Test: `apps/backend/internal/worker/link_discovery_test.go`
- Modify: `apps/backend/internal/worker/result_consumer.go`

**Requirements:**
- **Acceptance Criteria**
  1. Link discovery logic is isolated in a pure function.
  2. Logic handles exclusions, host matching, and depth checks.
  3. `ResultConsumer` uses this function.

- **Functional Requirements**
  1. Input: `links []string`, `currentDepth int`, `maxDepth int`, `exclusions []string`, `sourceID string`, `host string`.
  2. Output: `[]PageDTO`.

- **Non-Functional Requirements**
  - Pure function, no I/O.

- **Test Coverage**
  - [Unit] `TestDiscoverLinks` - various exclusion/depth scenarios.

**Step 1: Write failing test**
```go
package worker

import (
	"testing"
)

func TestDiscoverLinks(t *testing.T) {
	links := []string{
		"https://example.com/page1",
		"https://example.com/page2#frag",
		"https://other.com/page3",
		"https://example.com/exclude",
	}
	exclusions := []string{".*exclude.*"}
	
	pages := DiscoverLinks("src1", "example.com", links, 0, 2, exclusions)
	
	if len(pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].URL != "https://example.com/page1" {
		t.Errorf("expected page1, got %s", pages[0].URL)
	}
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL (Undefined DiscoverLinks)

**Step 3: Write minimal implementation**
```go
package worker

import (
	"net/url"
	"regexp"
)

func DiscoverLinks(sourceID, host string, links []string, currentDepth, maxDepth int, exclusions []string) []PageDTO {
	if currentDepth >= maxDepth {
		return nil
	}

	var newPages []PageDTO
	seen := make(map[string]bool)

	for _, link := range links {
		// 1. External Check
		linkU, err := url.Parse(link)
		if err != nil || linkU.Host != host {
			continue
		}

		// Normalize: Strip Fragment
		linkU.Fragment = ""
		normalizedLink := linkU.String()

		// 2. Exclusion Check
		excluded := false
		for _, ex := range exclusions {
			if matched, _ := regexp.MatchString(ex, normalizedLink); matched {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		if seen[normalizedLink] {
			continue
		}
		seen[normalizedLink] = true

		newPages = append(newPages, PageDTO{
			SourceID: sourceID,
			URL:      normalizedLink,
			Status:   "pending",
			Depth:    currentDepth + 1,
		})
	}
	return newPages
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS

### Task 4: Fixes - ResultConsumer Context & MCP Error

**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go`
- Modify: `apps/backend/features/mcp/handler.go`
- Test: `apps/backend/features/mcp/handler_test.go` (if exists)

**Requirements:**
- **Acceptance Criteria**
  1. `ResultConsumer` propagates context correctly (via `middleware.WithCorrelationID` and new Logger).
  2. `qurio_search` returns JSON-RPC error on internal failure.

- **Functional Requirements**
  1. `Handler.processRequest`: return `&JSONRPCResponse{Error: ...}` when search fails.
  2. `ResultConsumer`: Integrate `DiscoverLinks`.

- **Test Coverage**
  - Verify compile and logic flow.

**Step 3: Write minimal implementation (MCP)**
```go
// apps/backend/features/mcp/handler.go

// Inside processRequest for qurio_search
if err != nil {
    slog.Error("search failed", "error", err)
    // Return proper JSON-RPC error
    resp := makeErrorResponse(req.ID, ErrInternal, "Search failed: "+err.Error())
    return &resp
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/...`

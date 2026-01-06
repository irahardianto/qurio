---
name: technical-constitution
description: Generates technical implementation plans and architectural strategies that enforce the Project Constitution.
---

# Implementation Plan - Backend Test Coverage Boost

**Scope:** comprehensive unit test coverage for backend core components (`main.go` wiring, `mcp`, `worker`, `adapters`) to reach 95% target.

**Gap Analysis:**
- **App Wiring:** `app.New` uses live structs; needs interface-based mocks (`apps/backend/internal/app/mocks_test.go`).
- **MCP Handler:** Missing negative paths and tool-specific edge cases (`apps/backend/features/mcp/handler_test.go`).
- **Worker:** `ResultConsumer` lacks "poison pill" and timeout tests (`apps/backend/internal/worker/result_consumer_test.go`).
- **Handlers:** `Source` and `Job` handlers missing 404/Validation paths.
- **Adapters:** Weaviate (Network errors) and Gemini (Key Rotation) coverage is low.

**Knowledge Enrichment:**
- **RAG Queries:**
  - "Go dependency injection test pattern" -> Confirmed interface-based mocking strategy.
  - "NSQ consumer testing pattern" -> Validated use of `nsq.Message` injection in tests.
  - "Table driven tests in Go" -> Standard pattern for `handler_test.go`.
- **Reference:** `apps/backend/internal/app/app.go`, `apps/backend/features/mcp/handler.go`.

---

### Task 1: Implement Core Mocks

**Files:**
- Modify: `apps/backend/internal/app/mocks_test.go`
- Test: `apps/backend/internal/app/app_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `MockDatabase`, `MockVectorStore`, `MockTaskPublisher` fully implemented.
  2. `app.New` can be initialized in tests without external connections.

- **Test Coverage**
  - [Unit] `MockDatabase` methods (Exec, Query, etc.)
  - [Unit] `MockVectorStore` methods (EnsureSchema, StoreChunk)

**Step 1: Write failing test**
```go
// apps/backend/internal/app/app_test.go
func TestNew_WithMocks(t *testing.T) {
    mockDB := &MockDatabase{} // Will fail compile if not defined
    app, err := New(cfg, mockDB, &MockVectorStore{}, &MockTaskPublisher{}, logger)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/app/... -v`
Expected: FAIL (undefined structs)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/app/mocks_test.go
package app

import (
    "context"
    "database/sql"
    "qurio/apps/backend/internal/worker"
    "qurio/apps/backend/internal/retrieval"
)

type MockDatabase struct {
    *sql.DB // Embed for interface compliance if needed, or implement methods
    // Better: Implement interface methods
}
// Implement Query, Exec, etc. to return nil or mock data

type MockVectorStore struct{}
func (m *MockVectorStore) EnsureSchema(ctx context.Context) error { return nil }
func (m *MockVectorStore) StoreChunk(ctx context.Context, c worker.Chunk) error { return nil }
// ... implement all methods

type MockTaskPublisher struct{}
func (m *MockTaskPublisher) Publish(topic string, body []byte) error { return nil }
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/app/... -v`
Expected: PASS

---

### Task 2: MCP Handler - Protocol & Tool List Tests

**Files:**
- Modify: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `initialize` returns correct capabilities.
  2. `tools/list` returns all 4 tools.
  3. `tools/call` with unknown tool returns error code -32601.

- **Test Coverage**
  - [Unit] `processRequest` (Table-driven)

**Step 1: Write failing test**
```go
// apps/backend/features/mcp/handler_test.go
func TestProcessRequest_Protocol(t *testing.T) {
    tests := []struct {
        name    string
        req     JSONRPCRequest
        wantErr int
    }{
        {
            name: "Unknown Tool",
            req:  JSONRPCRequest{Method: "tools/call", Params: json.RawMessage(`{"name": "unknown"}`)},
            wantErr: -32601,
        },
    }
    // ... test runner
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/... -v`
Expected: FAIL (if logic missing or test incorrect)

**Step 3: Write minimal implementation**
```go
// Ensure processRequest handles ErrMethodNotFound correctly (already implemented, just verifying coverage)
// If missing, add check:
if params.Name != "qurio_search" && ... {
    return &JSONRPCResponse{Error: map[string]interface{}{"code": -32601}}
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/... -v`
Expected: PASS

---

### Task 3: MCP Handler - Search & Read Tests

**Files:**
- Modify: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `qurio_search` handles invalid alpha/missing query.
  2. `qurio_read_page` handles 404/empty results.

- **Test Coverage**
  - [Unit] `qurio_search` edge cases.
  - [Unit] `qurio_read_page` edge cases.

**Step 1: Write failing test**
```go
// apps/backend/features/mcp/handler_test.go
func TestProcessRequest_SearchEdges(t *testing.T) {
    // Add test case for missing query -> ErrInvalidParams
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// apps/backend/features/mcp/handler.go
// Ensure validation logic exists
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/... -v`
Expected: PASS

---

### Task 4: Worker ResultConsumer - Poison Pill

**Files:**
- Modify: `apps/backend/internal/worker/result_consumer_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Invalid JSON does not crash worker.
  2. Returns nil (ack) to NSQ to drop bad message.

- **Test Coverage**
  - [Unit] `HandleMessage` with malformed body.

**Step 1: Write failing test**
```go
func TestHandleMessage_InvalidJSON(t *testing.T) {
    // ... setup
    msg := nsq.NewMessage(nsq.MessageID{}, []byte("{invalid-json"))
    err := consumer.HandleMessage(msg)
    if err != nil {
        t.Errorf("expected nil error (ack), got %v", err)
    }
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/worker/result_consumer.go
if err := json.Unmarshal(m.Body, &payload); err != nil {
    slog.Error("invalid message", "error", err)
    return nil // Explicitly return nil to Ack
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS

---

### Task 5: Worker ResultConsumer - Timeout & Errors

**Files:**
- Modify: `apps/backend/internal/worker/result_consumer_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Embedding failure triggers retry (returns error).
  2. Context timeout is respected.

- **Test Coverage**
  - [Unit] `HandleMessage` with MockEmbedder failing.

**Step 1: Write failing test**
```go
func TestHandleMessage_EmbedderError(t *testing.T) {
    mockEmbedder.ReturnError = true
    // ...
    err := consumer.HandleMessage(msg)
    if err == nil {
        t.Errorf("expected error for retry")
    }
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// Ensure error is returned in result_consumer.go
if err != nil {
    return err
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS

---

### Task 6: Source & Job Handler 404s

**Files:**
- Modify: `apps/backend/features/source/handler_test.go`
- Modify: `apps/backend/features/job/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `GET /sources/{id}` with unknown ID returns 404.
  2. `POST /jobs/{id}/retry` with unknown ID returns 404.

- **Test Coverage**
  - [Unit] `Get`
  - [Unit] `Retry`

**Step 1: Write failing test**
```go
// apps/backend/features/source/handler_test.go
func TestGet_NotFound(t *testing.T) {
    // Setup request with ID "999"
    // Expect StatusNotFound
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/source/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// apps/backend/features/source/handler.go
if err == sql.ErrNoRows {
    writeError(w, http.StatusNotFound, "NOT_FOUND", "Source not found")
    return
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/source/... -v`
Expected: PASS

---

### Task 7: Weaviate Adapter Network Errors

**Files:**
- Modify: `apps/backend/internal/adapter/weaviate/store_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. 503 Service Unavailable is handled.
  2. GraphQL errors are reported.

- **Test Coverage**
  - [Unit] `Search` with httptest server returning 503.

**Step 1: Write failing test**
```go
func TestSearch_NetworkError(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusServiceUnavailable)
    }))
    defer ts.Close()
    // Configure client to use ts.URL
    // Expect error
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/adapter/weaviate/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/adapter/weaviate/store.go
// Ensure client checks status code or error
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/adapter/weaviate/... -v`
Expected: PASS

---

### Task 8: Gemini Key Rotation

**Files:**
- Modify: `apps/backend/internal/adapter/gemini/embedder_test.go` (or `dynamic_test.go`)

**Requirements:**
- **Acceptance Criteria**
  1. `Embed` calls `SettingsService.Get` to retrieve latest key.

- **Test Coverage**
  - [Unit] `Embed` with MockSettingsService.

**Step 1: Write failing test**
```go
func TestEmbed_RotatesKey(t *testing.T) {
    // Setup mock settings to return Key A then Key B
    // Verify client uses Key B on second call
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/adapter/gemini/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/adapter/gemini/dynamic_embedder.go
// Ensure Get(ctx) is called inside Embed()
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/adapter/gemini/... -v`
Expected: PASS

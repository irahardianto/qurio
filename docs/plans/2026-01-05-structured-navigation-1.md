### Task 1: Refactor MCP Handler Dependencies

**Files:**
- Modify: `apps/backend/features/mcp/handler.go`
- Modify: `apps/backend/main.go`
- Modify: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `mcp.NewHandler` accepts a `SourceManager` interface.
  2. `main.go` successfully compiles with the new dependency injection.
  3. `mcp.Handler` has access to source listing and page listing methods via the interface.

- **Functional Requirements**
  1. Define `SourceManager` interface in `mcp` package matching `source.Service` signatures for `List` and `GetPages`.

- **Non-Functional Requirements**
  - Maintain backward compatibility for existing `retrieval` dependency.

- **Test Coverage**
  - [Unit] `NewHandler` assigns dependencies correctly.
  - [Unit] Mock `SourceManager` implementation in `handler_test.go`.

**Step 1: Write failing test**
In `apps/backend/features/mcp/handler_test.go`:
```go
// Add a test that tries to construct a Handler with a SourceManager mock
// This will fail compilation first, which is valid TDD for signature changes in Go
func TestNewHandler_WithSourceManager(t *testing.T) {
    mockRetriever := &MockRetriever{}
    mockSourceMgr := &MockSourceManager{} // Needs to be defined
    
    // This function signature doesn't exist yet
    h := NewHandler(mockRetriever, mockSourceMgr)
    
    if h == nil {
        t.Fatal("Handler should not be nil")
    }
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/...`
Expected: FAIL (Compilation error: too many arguments in call to NewHandler)

**Step 3: Write minimal implementation**
In `apps/backend/features/mcp/handler.go`:
```go
// 1. Define Interface
type SourceManager interface {
    List(ctx context.Context) ([]source.Source, error) // Need to import source package or define local types?
    // Better to define local types or use interface{} to avoid circular dependency if source imports mcp?
    // source imports mcp? No. mcp imports retrieval. 
    // source does NOT import mcp. So we can import "qurio/apps/backend/features/source" in mcp.
    GetPages(ctx context.Context, id string) ([]source.SourcePage, error)
}

// 2. Update Struct
type Handler struct {
    retriever    Retriever
    sourceMgr    SourceManager // Add this
    sessions     map[string]chan string
    sessionsLock sync.RWMutex
}

// 3. Update Constructor
func NewHandler(r Retriever, s SourceManager) *Handler {
    return &Handler{
        retriever: r,
        sourceMgr: s,
        sessions:  make(map[string]chan string),
    }
}
```
In `apps/backend/main.go`:
```go
// Update call
mcpHandler := mcp.NewHandler(retrievalService, sourceService)
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/...`
Expected: PASS

---

### Task 2: Implement `qurio_list_sources` Tool

**Files:**
- Modify: `apps/backend/features/mcp/handler.go`
- Test: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `tools/list` returns `qurio_list_sources` in the list.
  2. Calling `qurio_list_sources` returns a JSON list of sources with `id`, `name`, and `type`.
  3. Returns appropriate empty message if no sources exist.

- **Functional Requirements**
  1. Map `source.Source` domain objects to a simplified JSON output.
  2. Handle context cancellation and errors from `SourceManager`.

- **Non-Functional Requirements**
  - Response time < 200ms (db query).

- **Test Coverage**
  - [Unit] `processRequest` with `tools/call` for `qurio_list_sources` returns correct JSON.
  - [Unit] Error handling when `sourceMgr.List` fails.

**Step 1: Write failing test**
```go
func TestHandle_ListSources(t *testing.T) {
    // Arrange
    mockSrc := &MockSourceManager{
        Sources: []source.Source{{ID: "src_1", Name: "Docs", Type: "web"}},
    }
    h := NewHandler(&MockRetriever{}, mockSrc)
    
    req := JSONRPCRequest{
        Method: "tools/call",
        Params: json.RawMessage(`{"name": "qurio_list_sources", "arguments": {}}`),
        ID:     1,
    }
    
    // Act
    resp := h.processRequest(context.Background(), req)
    
    // Assert
    // Check result contains "src_1" and "Docs"
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/...`
Expected: FAIL (Method not found or tool not implemented)

**Step 3: Write minimal implementation**
In `apps/backend/features/mcp/handler.go`:
1. Add `qurio_list_sources` to `tools/list` response.
2. Add handling logic in `tools/call`:
```go
if params.Name == "qurio_list_sources" {
    sources, err := h.sourceMgr.List(ctx)
    if err != nil {
        // handle error
    }
    // Format as JSON string
    // Return result
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/...`
Expected: PASS

---

### Task 3: Implement `qurio_list_pages` Tool

**Files:**
- Modify: `apps/backend/features/mcp/handler.go`
- Test: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `tools/list` includes `qurio_list_pages`.
  2. Calling `qurio_list_pages` with `source_id` returns list of pages.
  3. Validate `source_id` is required.

- **Functional Requirements**
  1. Input: `{ "source_id": "uuid" }`
  2. Output: List of `{ "url": "...", "status": "..." }`.

- **Test Coverage**
  - [Unit] Call with valid `source_id` returns pages.
  - [Unit] Call with missing `source_id` returns error.

**Step 1: Write failing test**
```go
func TestHandle_ListPages(t *testing.T) {
    // Arrange
    mockSrc := &MockSourceManager{
        Pages: map[string][]source.SourcePage{
            "src_1": {{URL: "/home", Status: "completed"}},
        },
    }
    h := NewHandler(&MockRetriever{}, mockSrc)
    
    // Act
    req := JSONRPCRequest{
        Method: "tools/call",
        Params: json.RawMessage(`{"name": "qurio_list_pages", "arguments": {"source_id": "src_1"}}`),
        ID: 1,
    }
    resp := h.processRequest(context.Background(), req)
    
    // Assert
    // Check result contains "/home"
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/...`
Expected: FAIL

**Step 3: Write minimal implementation**
In `apps/backend/features/mcp/handler.go`:
1. Add to `tools/list`.
2. Add logic to `tools/call`.
3. Parse `source_id`, call `h.sourceMgr.GetPages`.
4. Format output.

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/...`
Expected: PASS

---

### Task 4: Update `qurio_search` with Source Filtering

**Files:**
- Modify: `apps/backend/features/mcp/handler.go`
- Test: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `qurio_search` input schema includes optional `source_id`.
  2. When `source_id` is provided, it is passed to `retriever.Search` options.

- **Functional Requirements**
  1. Update `SearchArgs` struct.
  2. Map `source_id` arg to `Filters["sourceId"]`.

- **Test Coverage**
  - [Unit] `qurio_search` with `source_id` sets correct Filter options in `Retriever` call.

**Step 1: Write failing test**
In `apps/backend/features/mcp/handler_test.go`:
```go
func TestHandle_Search_WithSourceID(t *testing.T) {
    mockRetriever := &MockRetriever{
        SearchFunc: func(ctx context.Context, query string, opts *retrieval.SearchOptions) ([]retrieval.SearchResult, error) {
            if opts.Filters["sourceId"] != "src_123" {
                t.Errorf("Expected sourceId filter 'src_123', got %v", opts.Filters["sourceId"])
            }
            return []retrieval.SearchResult{}, nil
        },
    }
    h := NewHandler(mockRetriever, &MockSourceManager{})
    
    req := JSONRPCRequest{
        Method: "tools/call",
        Params: json.RawMessage(`{"name": "qurio_search", "arguments": {"query": "test", "source_id": "src_123"}}`),
        ID: 1,
    }
    
    h.processRequest(context.Background(), req)
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/...`
Expected: FAIL (Filter not set or argument ignored)

**Step 3: Write minimal implementation**
In `apps/backend/features/mcp/handler.go`:
1. Update `SearchArgs` struct: `SourceID *string json:"source_id,omitempty"`.
2. Update `tools/list` schema description.
3. In `processRequest`, if `args.SourceID` is set, add to `opts.Filters["sourceId"]`.

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/...`
Expected: PASS

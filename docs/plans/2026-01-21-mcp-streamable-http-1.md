# Implementation Plan - Refactor MCP to Streamable HTTP

## Proposed Changes
Refactor `apps/backend/features/mcp` to use a stateless, single-connection HTTP streaming transport, replacing the complex SSE+POST model. This aligns with the "Simplicity" and "Reliability" directives of the Technical Constitution.

## Tasks

### Task 1: Refactor MCP Handler Core
**Files:**
- Modify: `apps/backend/features/mcp/handler.go`
- Modify: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `ServeHTTP` accepts `POST` requests and sets `Transfer-Encoding: chunked`.
  2. Handler reads multiple JSON-RPC requests sequentially from a single request body stream.
  3. Handler writes JSON-RPC responses to the response body stream using a shared `json.Encoder`.
  4. Handler flushes output after each response using `http.Flusher`.
  5. `sessions` map, `sessionsLock`, `HandleSSE`, and `HandleMessage` are removed.
  6. **Critical:** Error responses (e.g., JSON parse errors) must be encoded into the stream without calling `w.WriteHeader` again.

- **Test Coverage**
  - [Unit] `TestServeHTTP_Streaming`: Simulate a request body with 2+ JSON objects, verify 2+ JSON responses are flushed sequentially.
  - [Unit] `TestServeHTTP_Streaming_Error`: Simulate a stream with valid JSON followed by malformed JSON. Verify valid response is flushed, followed by error response/termination.
  - [Unit] `TestServeHTTP_Context_Propagation`: Verify `CorrelationID` from request context is available during request processing.

**Step 1: Write failing tests**
In `apps/backend/features/mcp/handler_test.go`:
```go
func TestServeHTTP_Streaming(t *testing.T) {
	// ... (Existing happy path test)
}

func TestServeHTTP_Streaming_Error(t *testing.T) {
	// Arrange
	handler := mcp.NewHandler(&mockRetriever{}, &mockSourceMgr{})
	// Valid Request + Malformed JSON
	reqBody := `{"jsonrpc":"2.0","method":"ping","id":1}{"jsonrpc":"2.0", "bad":` 
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	// Expect 1 valid response then stop (or error frame depending on implementation)
	// For this plan, we expect the handler to send an error frame on decode failure.
	decoder := json.NewDecoder(rec.Body)
	
	// 1. First response success
	var resp1 mcp.JSONRPCResponse
	if err := decoder.Decode(&resp1); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}
	if resp1.Error != nil {
		t.Errorf("unexpected error in first response: %v", resp1.Error)
	}

	// 2. Second response should be error
	var resp2 mcp.JSONRPCResponse
	if err := decoder.Decode(&resp2); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp2.Error == nil {
		t.Error("expected error response for malformed input, got success")
	}
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/mcp/handler_test.go`
Expected: Fail (likely parse error or only one response handled)

**Step 3: Write minimal implementation**
In `apps/backend/features/mcp/handler.go`:
```go
// ... (Previous implementation)
// Ensure decode error logic:
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			slog.Error("mcp decode error", "error", err)
			
			// Send error frame
			errResp := makeErrorResponse(nil, ErrParse, "Parse error")
			encoder.Encode(errResp)
			if ok { flusher.Flush() }
			return
		}
// ...
```

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/mcp/handler_test.go`


### Task 2: Cleanup Routes and Legacy Code
**Files:**
- Modify: `apps/backend/internal/app/app.go`
- Modify: `apps/backend/features/mcp/handler.go` (Verify deletion of legacy methods)

**Requirements:**
- **Acceptance Criteria**
  1. `/mcp/sse` and `/mcp/messages` routes are removed from `app.go`.
  2. `/mcp` route is registered to `mcpHandler.ServeHTTP` (wrapped in middleware).
  3. Application compiles without errors.

**Step 1: Write failing test**
(Compile check)
Run: `go build ./apps/backend/...`
Expected: Fail if `app.go` references deleted methods `HandleSSE`/`HandleMessage`.

**Step 2: Verify test fails**
(See above)

**Step 3: Write minimal implementation**
In `apps/backend/internal/app/app.go`:
```go
	// Feature: Retrieval & MCP
	// ...
	mcpHandler := mcp.NewHandler(retrievalService, sourceService)
	
	// Unified Endpoint
	// Note: mcpHandler itself satisfies http.Handler interface now if ServeHTTP is defined on *Handler
	mux.Handle("/mcp", middleware.CorrelationID(enableCORS(mcpHandler.ServeHTTP))) 
	
	// DELETE these lines:
	// mux.Handle("GET /mcp/sse", ...)
	// mux.Handle("POST /mcp/messages", ...)
```

**Step 4: Verify test passes**
Run: `go build ./apps/backend/...`


### Task 3: Update Integration Tests
**Files:**
- Modify: `apps/backend/features/mcp/handler_integration_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `TestSSE_Correlation` is removed.
  2. `TestIntegration_Streaming_Correlation` is added: Verifies `X-Correlation-ID` header is preserved in the request context and logged (mock verification or response echo if possible).
  3. Integration tests verify that `POST /mcp` handles valid JSON-RPC calls end-to-end.

**Step 1: Write failing test**
In `apps/backend/features/mcp/handler_integration_test.go`:
```go
func TestIntegration_Streaming_Correlation(t *testing.T) {
	// Arrange
	// Setup handler with a mock/spy retrieval service that captures context
	spyRetriever := &spyRetriever{} 
	handler := mcp.NewHandler(spyRetriever, &mockSourceMgr{})
	
	correlationID := "test-correlation-id"
	reqBody := `{"jsonrpc":"2.0","method":"qurio_search","params":{"name":"qurio_search","arguments":{"query":"test"}},"id":1}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(reqBody))
	req.Header.Set("X-Correlation-ID", correlationID)
	
	// Apply Middleware (simulating app.go)
	mw := middleware.CorrelationID(handler.ServeHTTP)
	rec := httptest.NewRecorder()

	// Act
	mw.ServeHTTP(rec, req)

	// Assert
	if spyRetriever.capturedCtx == nil {
		t.Fatal("retriever was not called")
	}
	
	// Check if CorrelationID is in context
	// Assuming middleware.GetCorrelationID(ctx) works
	gotID := middleware.GetCorrelationID(spyRetriever.capturedCtx)
	if gotID != correlationID {
		t.Errorf("expected correlation ID %q, got %q", correlationID, gotID)
	}
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/mcp/handler_integration_test.go`
Expected: Fail (Compilation errors due to missing SSE methods or new test structure)

**Step 3: Write minimal implementation**
In `apps/backend/features/mcp/handler_integration_test.go`:
- Remove tests referencing `HandleSSE` or `HandleMessage`.
- Implement `spyRetriever` struct.
- Add `TestIntegration_Streaming_Correlation`.

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/mcp/handler_integration_test.go`

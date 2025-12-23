# Implementation Plan - MVP Part 3.5: Technical Compliance & Stabilization

**Ref:** `2025-12-23-qurio-mvp-part3-5`
**Feature:** Technical Compliance (Logging, Errors, Timeouts)
**Status:** Planned

## 1. Scope
Address critical architectural violations identified in `docs/2025-12-22-bugs-inconsistencies.md`. Specifically, enforce structured logging (structlog/slog), JSON error envelopes, and strict I/O timeouts across the Backend and Ingestion Worker.

**Gap Analysis:**
- **Backend (MCP):** Uses `http.Error` (text/plain) instead of JSON envelope. Missing request-scoped Correlation IDs in some handlers.
- **Worker (Logging):** Uses standard `logging` instead of `structlog` (JSON).
- **Worker (Reliability):** `Docling` and `Crawl4AI` executions lack explicit timeouts, risking indefinite hangs.

## 2. Requirements

### Functional
- **Error Responses:** All HTTP 4xx/5xx responses MUST return a JSON object with `status`, `error.code`, `error.message`, and `correlationId`.
- **Logging:** All logs MUST be structured JSON (in production) or pretty-printed (in dev), including `correlationId`, `level`, and `timestamp`.
- **Timeouts:** All external I/O (Crawling, Document Conversion) MUST hard-timeout after 60 seconds.

### Non-Functional
- **Observability:** Logs must be machine-parsable for future aggregation.
- **Reliability:** Worker must not hang indefinitely on a single task.

## 3. Tasks

### Task 1: Backend MCP Error Compliance
**Files:**
- Modify: `apps/backend/features/mcp/handler.go`
- Test: `apps/backend/features/mcp/handler_test.go` (Create if missing or modify)

**Requirements:**
- **Acceptance Criteria**
  1. `HandleMessage` returns JSON error for missing session/invalid JSON.
  2. `HandleMessage` generates and logs `correlationId`.
  3. `processRequest` logs include `correlationId` passed from handler.

- **Functional Requirements**
  1. Replace `http.Error(w, ...)` with `writeError(w, ...)` using JSON structure.
  2. Extract `writeError` to be reusable or use existing pattern.
  3. Generate `correlationId` at start of `HandleMessage` and `ServeHTTP`.

**Step 1: Write failing test**
```go
// apps/backend/features/mcp/handler_test.go
func TestHandleMessage_ErrorJSON(t *testing.T) {
    // Setup Handler
    // Request with missing sessionId
    // Assert status 400
    // Assert Content-Type application/json
    // Assert Body contains "error": {"code": ...}
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/...`

**Step 3: Implementation**
```go
// handler.go
func (h *Handler) HandleMessage(w http.ResponseWriter, r *http.Request) {
    correlationID := uuid.New().String()
    slog.Info("mcp message received", "correlation_id", correlationID, ...)

    // ... checks ...
    if sessionID == "" {
       h.writeError(w, nil, ErrInvalidParams, "Missing sessionId") // Update writeError to handle nil ID or separate HTTP error helper
       return
    }
}
// Add/Update writeHttpError for standard HTTP errors (not JSON-RPC responses)
func (h *Handler) writeHttpError(w http.ResponseWriter, code string, msg string, status int, correlationID string) {
    // JSON envelope
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/...`

### Task 2: Worker Logging Infrastructure
**Files:**
- Modify: `apps/ingestion-worker/requirements.txt`
- Create: `apps/ingestion-worker/logger.py`
- Modify: `apps/ingestion-worker/main.py`

**Requirements:**
- **Acceptance Criteria**
  1. `structlog` is installed.
  2. `main.py` initializes structured logging.
  3. Logs output as JSON strings.

- **Functional Requirements**
  1. Add `structlog` to requirements.
  2. Create `logger.py` to configure `structlog` (JSON renderer).
  3. Update `main.py` to use `structlog.get_logger()`.

**Step 1: Implementation**
```python
# requirements.txt
structlog
colorama # for dev pretty printing

# logger.py
import structlog
import logging
import sys

def configure_logger():
    # Configure structlog to wrap standard logging
    # Set JSON renderer
```

**Step 2: Verify implementation**
Run: `python3 apps/ingestion-worker/main.py` (Check stdout for JSON logs)

### Task 3: Worker Handlers Compliance (Log & Timeout)
**Files:**
- Modify: `apps/ingestion-worker/handlers/file.py`
- Modify: `apps/ingestion-worker/handlers/web.py`

**Requirements:**
- **Acceptance Criteria**
  1. Handlers use `structlog`.
  2. `handle_file_task` times out after 60s.
  3. `handle_web_task` times out after 60s.
  4. Timeouts are logged as errors.

- **Functional Requirements**
  1. Import `structlog`.
  2. Wrap `converter.convert` (in executor) with `asyncio.wait_for`.
  3. Wrap `crawler.arun` with `asyncio.wait_for`.
  4. Catch `asyncio.TimeoutError` and raise/log appropriately.

**Step 1: Implementation**
```python
# handlers/file.py
import structlog
import asyncio
logger = structlog.get_logger(__name__)

async def handle_file_task(...):
    try:
        result = await asyncio.wait_for(
            loop.run_in_executor(...),
            timeout=60.0
        )
    except asyncio.TimeoutError:
        logger.error("docling_conversion_timeout", path=file_path)
        raise
```

**Step 2: Verify implementation**
Run: `pytest apps/ingestion-worker/tests/test_handlers.py`

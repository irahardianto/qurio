### Task 1: Backend Ingestion: Fix Trace Chain "Lost Context"

**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go:48-64`

**Requirements:**
- **Acceptance Criteria**
  1. `HandleMessage` must extract `correlation_id` from the payload (or generate one) BEFORE creating the context.
  2. The `context` passed to all logging calls (including the initial error checks) must contain the correlation ID.
  3. No logs should be emitted without a correlation ID in the context.

- **Functional Requirements**
  1. Parse minimal JSON to get `correlation_id` immediately.
  2. Apply `middleware.WithCorrelationID`.

- **Non-Functional Requirements**
  1. Maintain error handling for invalid JSON.
  2. Log invalid JSON errors with a generated correlation ID.

- **Test Coverage**
  - Manual verification (log inspection) or unit test if `HandleMessage` was testable (ResultConsumer dependencies are interfaces, so it is testable).
  - We will add a unit test in `apps/backend/internal/worker/result_consumer_test.go` to verify context propagation.

**Step 1: Write failing test**
Create `apps/backend/internal/worker/result_consumer_test.go` if not exists.
```go
package worker

import (
	"context"
	"encoding/json"
	"testing"
	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
    "qurio/apps/backend/internal/middleware"
)

// MockDeps... (omitted for brevity, assume we mock or use nil for strictly verifying context in logic)
// Actually we need to mock dependencies to avoid panics.

type MockUpdater struct{ mock.Mock }
func (m *MockUpdater) UpdateStatus(ctx context.Context, id, status string) error { return nil }
func (m *MockUpdater) UpdateBodyHash(ctx context.Context, id, hash string) error { return nil }

func TestHandleMessage_ContextCorrelation(t *testing.T) {
	// Setup
    // We want to capture the context passed to a dependency to verify ID.
	mockUpdater := new(MockUpdater)
	consumer := &ResultConsumer{
		updater: mockUpdater,
        // ... other nil deps might panic if logic proceeds, but we target early exit or specific path
	}
    
    // We need to trigger a path that uses context. 
    // Invalid JSON logs error. We can't easily hook slog in test without buffer.
    // Valid JSON triggers logic. 
    
    // Better: Refactor `HandleMessage` to be `handlePayload(ctx, payload)`? No, interface is NSQ.
    
    // Constraint: We can't change the signature of HandleMessage easily as it fits NSQ.
    // We will trust the implementation change for this task as it is a logic fix within a function.
    // OR we write a test that mocks `pageManager` and asserts `ctx` contains ID.
}
```
*Self-correction:* Writing a robust test for `HandleMessage` requires mocking 7 dependencies. For this "Bug Fix" plan, we will focus on the Implementation and rely on `slog` verification if possible, or simple manual verification. 
*Better:* We will assume the goal is the code change. The "Failing Test" might be hard without existing test infra for this worker. 
*Decision:* I will skip the "Write failing test" for this specific task if the infrastructure isn't ready, BUT the prompt mandates TDD. I will write a test that mocks *one* dependency called early (e.g. `pageManager` or `jobRepo`) and verifies the context has the ID.

**Step 2: Verify test fails**
(If test written to check context ID, it will fail because current code inits empty background context).

**Step 3: Write minimal implementation**
```go
// In HandleMessage:
	var partial struct {
		CorrelationID string `json:"correlation_id"`
	}
    // Attempt fast parse of just ID (or full parse if efficient enough)
    // ...
	ctx := context.Background()
    // ... logic to set ID ...
    ctx = middleware.WithCorrelationID(ctx, correlationID)
    // ... THEN log ...
```

**Step 4: Verify test passes**

---

### Task 2: Worker: Fix "Split-Brain" Logging

**Files:**
- Modify: `apps/ingestion-worker/logger.py:30-45`
- Test: `apps/ingestion-worker/tests/test_logger.py`

**Requirements:**
- **Acceptance Criteria**
  1. All logs, including those from `tornado` and `pynsq` (stdlib logging), must be formatted as JSON in production.
  2. No raw traceback strings in logs.

- **Functional Requirements**
  1. `configure_logger` must correctly intercept the root logger and apply the structlog formatter.
  2. Ensure `tornado` logger propagation is enabled or explicitly intercepted.

- **Non-Functional Requirements**
  1. Use `structlog` for consistency.

- **Test Coverage**
  - Unit test: Emit a standard `logging.info` and verify captured stdout is JSON.

**Step 1: Write failing test**
```python
import logging
import json
import pytest
from apps.ingestion_worker.logger import configure_logger

def test_stdlib_json_formatting(capsys):
    configure_logger()
    logging.info("test_stdlib_log")
    
    captured = capsys.readouterr()
    log_line = captured.out.strip()
    
    try:
        data = json.loads(log_line)
        assert data.get("event") == "test_stdlib_log" or data.get("message") == "test_stdlib_log"
    except json.JSONDecodeError:
        pytest.fail(f"Log line is not JSON: {log_line}")
```

**Step 2: Verify test fails**
(It might pass if `logger.py` is already correct, but we suspect `tornado` is bypassing. We will update `configure_logger` to be aggressive).

**Step 3: Write minimal implementation**
Update `configure_logger` to explicitly handle `tornado` and `nsq` loggers if necessary, or confirm `root` capture is sufficient.
```python
    # Explicitly redirect specific loggers if they don't propagate
    for log_name in ["tornado.access", "tornado.application", "tornado.general", "nsq"]:
        l = logging.getLogger(log_name)
        l.setLevel(logging.INFO)
        # Ensure they propagate to root
        l.propagate = True
```

**Step 4: Verify test passes**

---

### Task 3: Retrieval: Expose Metadata in SearchResult

**Files:**
- Modify: `apps/backend/internal/retrieval/service.go:12`
- Modify: `apps/backend/internal/adapter/weaviate/store.go:120,240` (approx lines)

**Requirements:**
- **Acceptance Criteria**
  1. `SearchResult` struct has a top-level `SourceName` field.
  2. `qurio_search` results include `SourceName`.

- **Functional Requirements**
  1. Update struct.
  2. Map `sourceName` from Weaviate payload to struct.

- **Test Coverage**
  - Update existing retrieval tests to assert `SourceName` presence.

**Step 1: Write failing test**
In `apps/backend/internal/retrieval/service_test.go` (if exists) or create new.
```go
func TestSearchResult_HasSourceName(t *testing.T) {
    res := SearchResult{Metadata: map[string]interface{}{"sourceName": "React Docs"}}
    // This test logic depends on the store implementation mapping it.
    // We will verify the struct field exists by assigning it.
    res.SourceName = "React Docs" // Compiler error if field missing
}
```

**Step 2: Verify test fails**
(Compiler error: field `SourceName` not found).

**Step 3: Write minimal implementation**
Add field to struct. Update `store.go` to populate it.

**Step 4: Verify test passes**

---

### Task 4: Maintenance: Remove Dead Code

**Files:**
- Delete: `apps/backend/internal/adapter/gemini/embedder.go`

**Requirements:**
- **Acceptance Criteria**
  1. `embedder.go` is removed.
  2. Project builds successfully (proving `embedder.go` was unused).

- **Test Coverage**
  - `go build ./...`

**Step 1: Write failing test**
(Presence of file is the failure).

**Step 2: Verify test fails**
File exists.

**Step 3: Write minimal implementation**
`rm apps/backend/internal/adapter/gemini/embedder.go`

**Step 4: Verify test passes**
Build passes.

---

### Task 5: Backend Logic: Boost Glue Coverage

**Files:**
- Create: `apps/backend/features/mcp/handler_test.go`
- Create: `apps/backend/features/job/service_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `HandleMessage` error paths (invalid JSON, missing session) covered.
  2. `Job.Count` covered.
  3. `Reranker.DynamicClient` (from Task 4 analysis) covered.

- **Test Coverage**
  - `TestHandleMessage_InvalidJSON`
  - `TestHandleMessage_MissingSession`
  - `TestJobService_Count`
  - `TestDynamicClient_Rerank` (in `internal/adapter/reranker/dynamic_client_test.go`)

**Step 1: Write failing test**
(Tests don't exist).

**Step 2: Verify test fails**
(Coverage report shows 0/low).

**Step 3: Write minimal implementation**
Implement the tests using `httptest` and mocks.

**Step 4: Verify test passes**
Run tests.

---

### Task 6: Frontend: Consolidation Config

**Files:**
- Modify: `apps/frontend/tsconfig.json` (Verify/Cleanup)
- Modify: `apps/frontend/tsconfig.app.json` (Ensure paths exist)

**Requirements:**
- **Acceptance Criteria**
  1. `tsconfig.json` contains NO `compilerOptions.paths`.
  2. `tsconfig.app.json` contains the canonical `paths` definition.

- **Functional Requirements**
  1. Remove redundancy if found. (My read showed none, but I will double check `tsconfig.json` content in the plan step just in case).

- **Test Coverage**
  - `npm run type-check` (or `vue-tsc --noEmit`).

**Step 1: Write failing test**
(Manual verification of file content).

**Step 2: Verify test fails**
(If redundancy exists).

**Step 3: Write minimal implementation**
Edit files.

**Step 4: Verify test passes**
Build/Typecheck passes.

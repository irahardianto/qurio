# Implementation Plan - MVP Part 3.7: Bug Fixes & Standardization

**Ref:** `2025-12-23-bugs-inconsistencies.md`
**Status:** Planned
**Date:** 2025-12-23

## 1. Scope
Address critical technical debt, inconsistencies, and bugs identified in the project stability review. This covers API standardization, reliability (timeouts, tracing), and frontend/backend consistency.

**Gap Analysis:**
- **Tracing:** Correlation IDs are regenerated on error, breaking trace chains.
- **API:** Success responses lack `data/meta` envelope, inconsistent with error responses.
- **Reliability:** Worker loops lack timeouts for external calls.
- **Frontend:** Non-standard UI components (`<textarea>`) used.
- **Worker:** Inconsistent return types between handler implementations.

## 2. Requirements

### Functional
- **Correlation ID:** Every request MUST have a unique `X-Correlation-ID`. If missing, generate one. This ID MUST be used in logs and error responses.
- **API Standard:** All success responses MUST use `{ "data": ... }` envelope. Lists MUST include `{ "meta": { "count": N } }`.
- **Worker Timeouts:** Embedder and Vector Store operations MUST have a hard timeout (e.g., 60s).
- **Frontend:** `SourceForm` MUST use a standardized `Textarea` component matching the Design System.

### Non-Functional
- **Observability:** "Request Received" and "Request Completed" logs MUST be present for all public handlers (Settings, Sources).
- **Code Quality:** Worker handlers must share a common return contract (`List[Dict]`) to simplify the dispatcher logic.

## 3. Tasks

### Task 1: Backend Middleware & Tracing
**Files:**
- Create: `apps/backend/internal/middleware/correlation.go`
- Modify: `apps/backend/main.go`
- Modify: `apps/backend/features/source/handler.go`
- Modify: `apps/backend/internal/settings/handler.go`
- Test: `apps/backend/internal/middleware/correlation_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Middleware extracts `X-Correlation-ID` or generates UUID.
  2. Middleware logs "request received" and "request completed" (replacing manual logs in handlers).
  3. `writeError` uses the ID from context, DOES NOT generate a new one.

- **Test Coverage**
  - [Unit] `CorrelationMiddleware`: Verify ID is set in context and response header.
  - [Integration] `writeError`: Verify ID matches request header.

**Step 1: Write failing test**
```go
// apps/backend/internal/middleware/correlation_test.go
package middleware

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestCorrelationID(t *testing.T) {
    handler := CorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id, ok := r.Context().Value(CorrelationKey).(string)
        if !ok || id == "" {
            t.Error("correlation id missing from context")
        }
    }))

    req := httptest.NewRequest("GET", "/", nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)

    if w.Header().Get("X-Correlation-ID") == "" {
        t.Error("header missing")
    }
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/middleware/...` (Will fail as package doesn't exist)

**Step 3: Implementation**
```go
// apps/backend/internal/middleware/correlation.go
package middleware

import (
    "context"
    "log/slog"
    "net/http"
    "time"
    "github.com/google/uuid"
)

type key int
const CorrelationKey key = 0

func CorrelationID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get("X-Correlation-ID")
        if id == "" {
            id = uuid.New().String()
        }

        ctx := context.WithValue(r.Context(), CorrelationKey, id)
        w.Header().Set("X-Correlation-ID", id)

        slog.Info("request received", "method", r.Method, "path", r.URL.Path, "correlation_id", id)
        start := time.Now()

        next.ServeHTTP(w, r.WithContext(ctx))

        slog.Info("request completed", "method", r.Method, "path", r.URL.Path, "correlation_id", id, "duration", time.Since(start))
    })
}

func GetCorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(CorrelationKey).(string); ok {
        return id
    }
    return "unknown"
}
```

**Step 4: Integration (Manual)**
- Update `main.go` to wrap routes with `middleware.CorrelationID`.
- Update handlers to remove manual entry/exit logs.
- Update `writeError` to use `middleware.GetCorrelationID(r.Context())`.

### Task 2: API Response Standardization
**Files:**
- Modify: `apps/backend/features/source/handler.go`
- Modify: `apps/backend/internal/settings/handler.go`
- Test: `apps/backend/features/source/handler_test.go` (Update assertions)

**Requirements:**
- **Acceptance Criteria**
  1. `GET /sources` returns `{ "data": [...], "meta": {"count": N} }`.
  2. `GET /sources/{id}` returns `{ "data": {...} }`.
  3. `GET /settings` returns `{ "data": {...} }`.

**Step 1: Write failing test (Update existing test)**
Modify `apps/backend/features/source/handler_test.go` to assert JSON structure has `data` field.

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/source/...`

**Step 3: Implementation**
```go
// source/handler.go
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    // ...
    resp := map[string]interface{}{
        "data": sources,
        "meta": map[string]int{"count": len(sources)},
    }
    json.NewEncoder(w).Encode(resp)
}

// source/handler.go
func (h *Handler) Get(...) {
    // ...
    json.NewEncoder(w).Encode(map[string]interface{}{"data": detail})
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/source/...`

### Task 3: Worker Timeouts
**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go`

**Requirements:**
- **Acceptance Criteria**
  1. `Embed` call is wrapped in `context.WithTimeout(60s)`.
  2. `StoreChunk` call is wrapped in `context.WithTimeout(60s)`.

**Step 1: Implementation**
```go
// apps/backend/internal/worker/result_consumer.go
func (h *ResultConsumer) HandleMessage(m *nsq.Message) error {
    // ... inside loop ...
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    vector, err := h.embedder.Embed(ctx, c)
    // ...
    if err := h.store.StoreChunk(ctx, chunk); err != nil { ... }
}
```

### Task 4: Worker Return Types
**Files:**
- Modify: `apps/ingestion-worker/handlers/file.py`
- Modify: `apps/ingestion-worker/main.py`
- Test: `apps/ingestion-worker/tests/test_handlers.py`

**Requirements:**
- **Acceptance Criteria**
  1. `handle_file_task` returns `list[dict]` `[{"url": path, "content": ...}]`.
  2. `main.py` removes special handling for `file` type result parsing.

**Step 1: Implementation**
```python
# handlers/file.py
async def handle_file_task(file_path: str) -> list[dict]:
    # ... conversion ...
    return [{"url": file_path, "content": content}]
```

```python
# main.py
if task_type == 'web':
    results_list = await handle_web_task(...)
elif task_type == 'file':
    results_list = await handle_file_task(...)
# No manual list wrapping needed
```

### Task 5: Frontend Textarea Component
**Files:**
- Create: `apps/frontend/src/components/ui/textarea/Textarea.vue`
- Create: `apps/frontend/src/components/ui/textarea/index.ts`
- Modify: `apps/frontend/src/features/sources/SourceForm.vue`

**Requirements:**
- **Acceptance Criteria**
  1. `Textarea` component exists with standard `shadcn` styling.
  2. `SourceForm` uses `Textarea` instead of `<textarea>`.

**Step 1: Implementation**
```vue
<!-- apps/frontend/src/components/ui/textarea/Textarea.vue -->
<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { useVModel } from "@vueuse/core"
import { cn } from "@/lib/utils"

const props = defineProps<{
  defaultValue?: string | number
  modelValue?: string | number
  class?: HTMLAttributes["class"]
}>()

const emits = defineEmits<{
  (e: "update:modelValue", payload: string | number): void
}>()

const modelValue = useVModel(props, "modelValue", emits, {
  passive: true,
  defaultValue: props.defaultValue,
})
</script>

<template>
  <textarea v-model="modelValue" :class="cn('flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', props.class)" />
</template>
```

**Step 2: Integration**
Import and use `Textarea` in `SourceForm.vue`.

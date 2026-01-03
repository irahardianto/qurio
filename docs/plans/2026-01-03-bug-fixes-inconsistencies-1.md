# Implementation Plan - Bug Fixes & Inconsistencies

**Feature:** Bug Fixes Inconsistencies
**Date:** 2026-01-03
**Status:** Planned

## Requirements Analysis

### Scope
Fix 5 specific architectural and implementation inconsistencies identified in `docs/2026-01-03-bug-inconsistencies.md` to restore standard compliance.

### Gap Analysis
- **Nouns Mapped:**
  - `file handler` -> `apps/ingestion-worker/handlers/file.py`
  - `path field` -> `path` key in return dict
  - `SourceList.vue` -> `apps/frontend/src/features/sources/SourceList.vue`
  - `tsconfig.json` -> `apps/frontend/tsconfig.json`
  - `source.store.ts` -> `apps/frontend/src/features/sources/source.store.ts`
  - `Chunk interface` -> `Chunk` type in store
  - `Source Name` -> `sourceName` in Backend structs/Weaviate
  - `ResultConsumer` -> `apps/backend/internal/worker/result_consumer.go`
  - `StoreChunk` -> `apps/backend/internal/adapter/weaviate/store.go`
  - `HelloWorld.vue` -> `apps/frontend/src/components/HelloWorld.vue`

- **Verbs Mapped:**
  - `return path` -> Task 1
  - `store sourceName` -> Task 2
  - `remove component` -> Task 3
  - `fix imports` -> Task 4
  - `rename casing` -> Task 5

### Exclusions
- None. All identified inconsistencies are addressable.

---

## Tasks

### Task 1: Ingestion Worker - Add Path to File Handler

**Files:**
- Modify: `apps/ingestion-worker/handlers/file.py`
- Test: `apps/ingestion-worker/tests/test_handlers.py`

**Requirements:**
- **Acceptance Criteria**
  1. `handle_file_task` returns a dictionary containing a `path` key.
  2. The `path` value is equivalent to the filename (or breadcrumb if applicable).
- **Functional Requirements**
  1. Return `path` metadata to enable contextual embedding alignment in backend.
- **Non-Functional Requirements**
  1. Must match web handler output structure.

**Step 1: Write failing test**
```python
# apps/ingestion-worker/tests/test_handlers.py
# Add to existing test class or create new test
def test_handle_file_task_returns_path(self):
    # Mock task and file processing
    # ... setup code ...
    result = handle_file_task(task_payload)
    assert "path" in result
    assert result["path"] == "expected_filename.md"
```

**Step 2: Verify test fails**
Run: `pytest apps/ingestion-worker/tests/test_handlers.py`
Expected: FAIL with "KeyError: 'path'" or assertion error.

**Step 3: Write minimal implementation**
```python
# apps/ingestion-worker/handlers/file.py
def handle_file_task(task):
    # ... existing processing ...
    return {
        "content": content,
        "title": title,
        "path": filename,  # Add this line
        # ... other fields
    }
```

**Step 4: Verify test passes**
Run: `pytest apps/ingestion-worker/tests/test_handlers.py`
Expected: PASS

---

### Task 2: Backend - Store Source Name in Weaviate

**Files:**
- Modify: `apps/backend/internal/worker/types.go` (Add SourceName to Chunk)
- Modify: `apps/backend/internal/vector/schema.go` (Add sourceName property)
- Modify: `apps/backend/internal/adapter/weaviate/store.go` (Map SourceName to property)
- Modify: `apps/backend/internal/worker/result_consumer.go` (Populate SourceName)
- Test: `apps/backend/internal/worker/result_consumer_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `DocumentChunk` class in Weaviate has `sourceName` property.
  2. Chunks stored via `ResultConsumer` include the `sourceName`.
- **Functional Requirements**
  1. Enable filtering by Source Name in RAG queries.
- **Non-Functional Requirements**
  1. Backward compatibility for existing schema (Weaviate handles additions well).

**Step 1: Write failing test**
```go
// apps/backend/internal/worker/result_consumer_test.go
func TestResultConsumer_PopulatesSourceName(t *testing.T) {
    // Setup consumer with mock store
    // Process message with SourceName
    // Assert Store.StoreChunk called with Chunk.SourceName populated
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/...`
Expected: FAIL (field missing or zero value)

**Step 3: Write minimal implementation**
```go
// 1. types.go: Add SourceName string to Chunk struct
// 2. schema.go: Add Property{Name: "sourceName", DataType: []string{"text"}}
// 3. result_consumer.go: chunk.SourceName = payload.SourceName
// 4. store.go: properties["sourceName"] = chunk.SourceName
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/...`
Expected: PASS

---

### Task 3: Frontend - Remove Cruft (HelloWorld.vue)

**Files:**
- Delete: `apps/frontend/src/components/HelloWorld.vue`
- Verify: `apps/frontend/src/App.vue` (Ensure it's not imported)

**Requirements:**
- **Acceptance Criteria**
  1. `HelloWorld.vue` is removed.
  2. Application builds without errors.

**Step 1: Verify presence**
Run: `ls apps/frontend/src/components/HelloWorld.vue`
Expected: File exists.

**Step 2: Delete file**
Run: `rm apps/frontend/src/components/HelloWorld.vue`

**Step 3: Verify build**
Run: `npm run build --prefix apps/frontend`
Expected: PASS (if not used) or FAIL (if imported). If fail, remove import in App.vue.

---

### Task 4: Frontend - Fix Import Inconsistencies & Config

**Files:**
- Modify: `apps/frontend/src/features/sources/SourceList.vue`
- Modify: `apps/frontend/tsconfig.json` (Remove paths if redundant)
- Test: `apps/frontend/src/features/sources/SourceList.spec.ts`

**Requirements:**
- **Acceptance Criteria**
  1. All imports in `SourceList.vue` use `@/` alias where appropriate.
  2. `tsconfig.json` does not duplicate `tsconfig.app.json` paths.
- **Functional Requirements**
  1. Maintain build integrity.

**Step 1: Write failing test (Lint/Build)**
(Note: Hard to write a failing unit test for imports, relies on build/lint check)
Run: `grep "\.\./\.\./" apps/frontend/src/features/sources/SourceList.vue`
Expected: Matches (indicating relative paths)

**Step 2: Write minimal implementation**
```typescript
// SourceList.vue
// Change: import StatusBadge from '../../components/ui/StatusBadge.vue'
// To: import StatusBadge from '@/components/ui/StatusBadge.vue'
```
```json
// tsconfig.json
// Remove "paths" if fully covered by tsconfig.app.json reference
```

**Step 3: Verify fixes**
Run: `npm run build --prefix apps/frontend`
Expected: PASS

---

### Task 5: Frontend - Fix Store Casing

**Files:**
- Modify: `apps/frontend/src/features/sources/source.store.ts`
- Test: `apps/frontend/src/features/sources/source.store.spec.ts`

**Requirements:**
- **Acceptance Criteria**
  1. `Chunk` interface properties use `snake_case` (e.g., `chunk_index`, `source_id`).
  2. API mapping logic correctly maps backend JSON to new interface keys.
- **Functional Requirements**
  1. Align with backend API naming standards.

**Step 1: Write failing test (Compilation)**
```typescript
// apps/frontend/src/features/sources/source.store.spec.ts
// Update test to expect snake_case keys
it('should map API response to Chunk', () => {
    // ...
    expect(chunk.chunk_index).toBeDefined(); // Will fail if type is ChunkIndex
})
```

**Step 2: Verify test fails**
Run: `npm run test:unit --prefix apps/frontend`
Expected: FAIL (Compilation error or undefined check)

**Step 3: Write minimal implementation**
```typescript
// source.store.ts
export interface Chunk {
  chunk_index: number;
  source_id: string;
  // ...
}
// Update mapper function
```

**Step 4: Verify test passes**
Run: `npm run test:unit --prefix apps/frontend`
Expected: PASS

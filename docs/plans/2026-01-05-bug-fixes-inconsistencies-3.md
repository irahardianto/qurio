---
name: technical-constitution
description: Generates technical implementation plans and architectural strategies that enforce the Project Constitution.
---

# Implementation Plan - Bug Fixes & Inconsistencies (Part 3)

**Status**: Proposed
**Date**: 2026-01-05
**Scope**: Resolution of identified "Hybrid Data Casing" inconsistency between Backend and Frontend.

## 1. Requirements Extraction

### Scope
Standardize the `Source` data model to strictly follow `snake_case` conventions across the stack. Specifically, replace the `lastSyncedAt` (CamelCase) phantom field in the Frontend with `updated_at` (snake_case) and ensure the Backend exposes this field from the persistent storage.

### Gap Analysis
- **Noun**: `Source.updated_at` (Backend) - Currently exists in DB but not in Go Struct or API response. -> **Task 1**
- **Noun**: `Source.lastSyncedAt` (Frontend) - Currently exists in TS Interface but not in Backend API. -> **Task 2** (Rename to `updated_at`)

### Exclusions (Verified as Fixed/Invalid)
1.  **API Response Envelope**: `GetSettings` and `source.List` already implement `{ "data": ... }` envelope. (Verified in `apps/backend/internal/settings/handler.go`)
2.  **Janitor Orchestration**: `ResetStuckPages` is explicitly called in `main.go` via a background ticker. (Verified in `apps/backend/main.go`)
3.  **MCP SSE Trace Chain**: `HandleMessage` uses `context.WithoutCancel(r.Context())`, preserving correlation IDs. (Verified in `apps/backend/features/mcp/handler.go`)
4.  **Cruft & Redundancy**: `HelloWorld.vue` is absent, and `tsconfig` files show no path alias redundancy. (Verified)

## 2. Knowledge Enrichment

**Context Sources:**
- `apps/backend/features/source/source.go`: Go Struct definition.
- `apps/backend/features/source/repo.go`: SQL queries.
- `apps/frontend/src/features/sources/source.store.ts`: TypeScript Interface.

**RAG & Reference**:
- Confirmed `sources` table schema via `apps/backend/migrations/000001_init_schema.up.sql`: `updated_at` column exists.
- Confirmed strict `snake_case` preference in `Technical Constitution`.

## 3. Implementation Tasks

### Task 1: Backend - Expose `updated_at` in Source API

**Files:**
- Modify: `apps/backend/features/source/source.go`
- Modify: `apps/backend/features/source/repo.go`
- Test: `apps/backend/features/source/repo_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `Source` struct includes `UpdatedAt string` with JSON tag `updated_at`.
  2. `repo.Get` and `repo.List` return populated `UpdatedAt` timestamps.
- **Functional Requirements**
  1. API consumers receive the last modification time of a source.
- **Non-Functional Requirements**
  1. No breaking changes for other fields.
- **Test Coverage**
  - [Integration] `TestPostgresRepo_Save` / `TestPostgresRepo_Get`: Verify `UpdatedAt` is not empty.

**Step 1: Write failing test**
```go
// apps/backend/features/source/repo_test.go
// Add assertion to existing test or create new test
func TestPostgresRepo_Get_HasTimestamp(t *testing.T) {
    // ... setup repo ...
    src := &source.Source{
        URL: "http://example.com",
        // ...
    }
    err := repo.Save(ctx, src)
    require.NoError(t, err)
    
    got, err := repo.Get(ctx, src.ID)
    require.NoError(t, err)
    require.NotEmpty(t, got.UpdatedAt, "UpdatedAt should not be empty")
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/source/... -v`
Expected: FAIL (field undefined or empty)

**Step 3: Write minimal implementation**
```go
// apps/backend/features/source/source.go
type Source struct {
    // ...
    UpdatedAt   string   `json:"updated_at"`
    // ...
}

// apps/backend/features/source/repo.go
// Update List query:
query := `SELECT id, ..., updated_at FROM sources ...`
// Update List scan:
rows.Scan(..., &s.UpdatedAt)

// Update Get query:
query := `SELECT id, ..., updated_at FROM sources ...`
// Update Get scan:
row.Scan(..., &s.UpdatedAt)
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/source/... -v`
Expected: PASS

---

### Task 2: Frontend - Standardize Source Interface

**Files:**
- Modify: `apps/frontend/src/features/sources/source.store.ts`
- Test: `apps/frontend/src/features/sources/source.store.spec.ts`

**Requirements:**
- **Acceptance Criteria**
  1. `Source` interface uses `updated_at` instead of `lastSyncedAt`.
  2. All references to `lastSyncedAt` are updated.
- **Functional Requirements**
  1. UI can display the correct update time from the backend.
- **Non-Functional Requirements**
  1. Strict TypeScript type compliance.

**Step 1: Write failing test**
```typescript
// apps/frontend/src/features/sources/source.store.spec.ts
it('should map updated_at correctly', async () => {
    // ... mock fetch with updated_at in payload ...
    await store.fetchSources()
    expect(store.sources[0].updated_at).toBeDefined()
})
```

**Step 2: Verify test fails**
Run: `npm run test:unit apps/frontend/src/features/sources/source.store.spec.ts`
Expected: FAIL (property `updated_at` does not exist on type)

**Step 3: Write minimal implementation**
```typescript
// apps/frontend/src/features/sources/source.store.ts
export interface Source {
  // ...
  updated_at?: string // Replaces lastSyncedAt
  // ...
}
```

**Step 4: Verify test passes**
Run: `npm run test:unit apps/frontend/src/features/sources/source.store.spec.ts`
Expected: PASS

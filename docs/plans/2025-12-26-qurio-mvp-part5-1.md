---
name: technical-constitution
description: Implementation plan for MVP Part 5.1 (Admin Completeness & Cleanup).
---

# Implementation Plan - MVP Part 5.1: Admin Completeness & Cleanup

**Ref:** `2025-12-26-qurio-mvp-part5-1`
**Feature:** Dashboard, Failed Jobs (DLQ), Source Cleanup, Documentation
**Status:** Draft

## 1. Scope
Implement missing Admin UI features (Dashboard, Failed Jobs/DLQ) and ensure data consistency upon source deletion. Finally, create user documentation.

**Gap Analysis:**
- **Failed Jobs:** No `failed_jobs` table or UI. Failures are currently lost or just marked as "failed" on source.
- **Source Cleanup:** Deleting a source leaves orphaned chunks in Weaviate.
- **Dashboard:** No home page with system statistics.
- **Docs:** No `README.md` usage instructions.

## 2. Requirements

### Functional
- **Failed Jobs:** Store failed ingestion jobs with error details. Allow manual retry (re-queue).
- **Source Cleanup:** Hard-delete chunks from Weaviate when a source is deleted.
- **Dashboard:** Show counts (Sources, Documents, Failed Jobs) and system status.
- **Docs:** Provide clear setup and usage guide.

### Non-Functional
- **Performance:** Stats queries should be fast (count queries).
- **Reliability:** Job retry must be idempotent (re-publish to NSQ).

## 3. Tasks

### Task 1: Database Migration (Failed Jobs)
**Files:**
- Create: `apps/backend/migrations/000009_create_failed_jobs.up.sql`
- Create: `apps/backend/migrations/000009_create_failed_jobs.down.sql`

**Requirements:**
- Table `failed_jobs`: `id` (UUID), `source_id` (UUID), `handler` (string), `payload` (JSONB), `error` (text), `created_at` (timestamp), `retries` (int).

**Step 1: Write Migration**
```sql
CREATE TABLE failed_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    handler TEXT NOT NULL, -- 'web' or 'file'
    payload JSONB NOT NULL,
    error TEXT NOT NULL,
    retries INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Task 2: Job Domain & Repository
**Files:**
- Create: `apps/backend/features/job/job.go` (Structs)
- Create: `apps/backend/features/job/repo.go` (Interface & Postgres Impl)
- Test: `apps/backend/features/job/repo_test.go`

**Requirements:**
- `Job` struct mapping to DB table.
- `Repo` methods: `Save(ctx, job)`, `List(ctx, limit, offset)`, `Get(ctx, id)`, `Delete(ctx, id)`.

**Step 1: Write failing test**
Create `repo_test.go` that attempts to save and retrieve a job.

**Step 3: Implementation**
Standard Postgres implementation using `database/sql`.

### Task 3: Result Consumer - Save Failures
**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go`

**Requirements:**
- Inject `JobRepo` into `ResultConsumer`.
- In `HandleMessage`: if `payload.Status == "failed"`, call `JobRepo.Save`.

**Step 1: Implementation**
```go
// result_consumer.go
// ...
if payload.Status == "failed" {
    job := &job.Job{
        SourceID: payload.SourceID,
        Handler:  payload.Handler, // Need to ensure Handler is in ResultPayload or derive it
        Payload:  payload.OriginalPayload, // Worker needs to send this back
        Error:    payload.Error,
    }
    h.jobRepo.Save(ctx, job)
}
```
*Note: If `OriginalPayload` is missing from `ResultPayload`, add it to `ingestion-worker` first (Plan Part 5.2). For MVP, assume we construct minimal payload or just log error.*

### Task 4: Job Handler (API)
**Files:**
- Create: `apps/backend/features/job/handler.go`
- Test: `apps/backend/features/job/handler_test.go`
- Modify: `apps/backend/main.go` (Register routes)

**Requirements:**
- `GET /api/jobs/failed`: Return list of failed jobs.
- `POST /api/jobs/:id/retry`: Retrieve job, publish to `ingest.task` topic, delete from `failed_jobs`.

**Step 1: Write failing test**
Test `GET /jobs/failed` returns 200.

**Step 3: Implementation**
Use `nsq.Producer` to re-publish.

### Task 5: Weaviate Delete by SourceID
**Files:**
- Modify: `apps/backend/internal/worker/interfaces.go` (VectorStore interface)
- Modify: `apps/backend/internal/adapter/weaviate/store.go`
- Test: `apps/backend/internal/adapter/weaviate/store_test.go`

**Requirements:**
- Add `DeleteChunksBySourceID(ctx, sourceID)`.
- Implement using Weaviate Batch Delete (`where source_id = ID`).

**Step 1: Write failing test**
Insert chunks, delete by SourceID, verify gone.

**Step 3: Implementation**
Same pattern as `DeleteChunksByURL` but filter only on `source_id`.

### Task 6: Source Service - Delete Cleanup
**Files:**
- Modify: `apps/backend/features/source/source.go`

**Requirements:**
- In `Delete` method: Call `VectorStore.DeleteChunksBySourceID` BEFORE soft-deleting from DB.

**Step 1: Implementation**
```go
// source.go
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
    // 1. Clean Vector Store
    if err := s.vectorStore.DeleteChunksBySourceID(ctx, id.String()); err != nil {
        return err
    }
    // 2. Soft Delete DB
    return s.repo.Delete(ctx, id)
}
```

### Task 7: Stats API
**Files:**
- Create: `apps/backend/features/stats/handler.go`
- Modify: `apps/backend/main.go`

**Requirements:**
- `GET /api/stats`: Return `{ "sources": 10, "documents": 500, "failed_jobs": 2 }`.
- Inject `SourceRepo`, `DocRepo` (if exists, or count chunks?), `JobRepo`.
- *Simpler*: `SourceRepo.Count()`, `JobRepo.Count()`. For documents, maybe `VectorStore.Count()`? Or just track in DB? DB `documents` table exists? (Yes, from PRD).

**Step 1: Implementation**
Aggregate counts from repos.

### Task 8: Frontend - API & Dashboard
**Files:**
- Modify: `apps/frontend/src/features/sources/source.api.ts` (Add `getStats`, `getFailedJobs`, `retryJob`)
- Create: `apps/frontend/src/views/DashboardView.vue`
- Modify: `apps/frontend/src/router/index.ts` (Set `/` to Dashboard)

**Requirements:**
- Dashboard: 3 Cards (Sources, Documents, Failed Jobs).
- Recent Sources list.

**Step 1: Implementation**
Standard Vue/Tailwind layout.

### Task 9: Frontend - Jobs View
**Files:**
- Create: `apps/frontend/src/views/JobsView.vue`
- Modify: `apps/frontend/src/components/layout/Sidebar.vue` (Add Jobs link)

**Requirements:**
- Table listing failed jobs.
- "Retry" button for each.
- "Retry All" (Optional).

**Step 1: Implementation**
Call `getFailedJobs`, display list. On Retry click -> API call -> Refresh list.

### Task 10: Documentation
**Files:**
- Modify: `README.md`

**Requirements:**
- "Getting Started": `docker-compose up -d`.
- "Configuration": `.env` vars.
- "Architecture": Diagram/Description.
- "API Reference": Link to code or brief list.

**Step 1: Implementation**
Write clear Markdown.

# Parallel Crawling & Distributed Ingestion Implementation Plan

**Scope:** Transform ingestion from monolithic batch to distributed page-level parallel system.
**Reference:** `docs/plans/2025-12-26-parallel-crawling-refactor.md`

## Requirements Extraction

### Acceptance Criteria
- [ ] **Database:** `source_pages` table exists with `status`, `depth`, and `unique(source_id, url)`.
- [ ] **Worker:** Python worker handles **single** URL, returns content + links, does **not** recurse.
- [ ] **Backend:** Consumer uses `AddConcurrentHandlers`.
- [ ] **Backend:** Consumer extracts links from worker result, creates new `source_pages` (deduplicated), and enqueues new tasks.
- [ ] **API:** `GET /sources/{id}/pages` returns pagination status.

### Non-Functional Requirements
- **Idempotency:** Re-processing a page should not duplicate chunks or `source_pages`.
- **Concurrency:** Worker and Backend must handle >1 concurrent tasks safely.
- **Performance:** Bulk insert for discovered links to avoid N+1 DB inserts.

## Implementation Tasks

### Task 1: Database Migration (Source Pages)

**Files:**
- Create: `apps/backend/migrations/000010_create_source_pages.up.sql`
- Create: `apps/backend/migrations/000010_create_source_pages.down.sql`

**Requirements:**
- **Functional:** Track individual page status in crawl.
- **Schema:**
  ```sql
  CREATE TABLE source_pages (
      id UUID PRIMARY KEY,
      source_id UUID REFERENCES sources(id) ON DELETE CASCADE,
      url TEXT NOT NULL,
      status TEXT DEFAULT 'pending',
      depth INTEGER DEFAULT 0,
      error TEXT,
      created_at TIMESTAMPTZ DEFAULT NOW(),
      updated_at TIMESTAMPTZ DEFAULT NOW(),
      UNIQUE(source_id, url)
  );
  ```

**Step 1: Write failing test**
*Skipped for SQL migration files (validated by migration tool).*

**Step 2: Verify test fails**
*Skipped.*

**Step 3: Write minimal implementation**
Create the SQL files.

**Step 4: Verify test passes**
Run: `make migrate-up` (or equivalent shell command to apply migrations)
Verify: `psql -c "\d source_pages"`

---

### Task 2: Backend SourcePage Repository

**Files:**
- Create: `apps/backend/features/source/page_repo.go`
- Test: `apps/backend/features/source/page_repo_test.go`

**Requirements:**
- **Functional:** `CreatePage`, `BulkCreatePages` (ignore conflicts), `UpdatePageStatus`, `GetPagesBySourceID`.
- **Performance:** Use `ON CONFLICT DO NOTHING` for bulk creation of discovered links.

**Step 1: Write failing test**
```go
package source

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/google/uuid"
)

func TestPageRepo_BulkCreate(t *testing.T) {
    // Requires integration test setup with real DB
    // ... setup db ...
    repo := NewPageRepo(db)
    sourceID := uuid.New()
    // Assume source exists (create it in setup)
    
    pages := []SourcePage{
        {SourceID: sourceID, URL: "http://a.com", Depth: 1},
        {SourceID: sourceID, URL: "http://b.com", Depth: 1},
        {SourceID: sourceID, URL: "http://a.com", Depth: 1}, // Duplicate
    }
    
    err := repo.BulkCreateIgnoreConflicts(context.Background(), pages)
    assert.NoError(t, err)
    
    // Verify only 2 pages exist
    stored, _ := repo.GetPagesBySourceID(context.Background(), sourceID)
    assert.Equal(t, 2, len(stored))
}
```

**Step 3: Write minimal implementation**
Implement `PostgresPageRepo` with `BulkCreateIgnoreConflicts` using `INSERT ... ON CONFLICT (source_id, url) DO NOTHING`.

---

### Task 3: Python Worker Refactor (Single Page Mode)

**Files:**
- Modify: `apps/ingestion-worker/handlers/web.py`
- Modify: `apps/ingestion-worker/main.py`
- Test: `apps/ingestion-worker/tests/test_handlers.py`

**Requirements:**
- **Functional:** Remove `BFSDeepCrawlStrategy`. Crawl ONLY the target URL.
- **Output:** JSON result must include `links: string[]`.
- **Link Extraction:** Use `crawl4ai` result or parse HTML to find `<a href="...">`.

**Step 1: Write failing test**
Modify `test_handlers.py` to assert that `handle_web_task` returns a dictionary with `links` key and does NOT recurse (mock the crawler to return HTML with links, assert it doesn't call itself).

**Step 3: Write minimal implementation**
- Remove recursion logic.
- Add link extraction (if not provided by crawler, use `BeautifulSoup` or regex on `result.html`).
- Return `{ "content": ..., "links": [...] }`.

---

### Task 4: Backend Result Consumer (Link Discovery)

**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go`
- Test: `apps/backend/internal/worker/result_consumer_test.go`

**Requirements:**
- **Functional:**
  1. Parse `links` from message.
  2. If `depth < max_depth`:
     - Filter external links (check domain).
     - Call `repo.BulkCreateIgnoreConflicts`.
     - For each *newly created* page (this is tricky with "Ignore Conflicts", maybe "On Conflict Do Nothing" returns rows affected? Or we blindly enqueue? Better: `INSERT ... ON CONFLICT DO NOTHING RETURNING url`. Only enqueue the returned URLs).
     - Publish new NSQ tasks for new pages.
  3. Mark current page `completed`.

**Step 1: Write failing test**
Unit test `HandleMessage`:
- Mock `PageRepo` and `NSQProducer`.
- Input: Message with `url="http://root.com"`, `links=["/sub1"]`, `depth=0`.
- Expect:
  - `repo.BulkCreate` called with `/sub1`.
  - `nsq.Publish` called with `/sub1` payload (depth 1).
  - `repo.UpdateStatus` called for root.

**Step 3: Write minimal implementation**
Implement the logic. Ensure `RETURNING` clause is used in Repo to identify which links are actually new, to avoid infinite loops or redundant queues.

---

### Task 5: Backend Concurrency Configuration

**Files:**
- Modify: `apps/backend/main.go`

**Requirements:**
- **Functional:** Change `consumer.AddHandler` to `consumer.AddConcurrentHandlers(handler, concurrency)`.
- **Config:** Read concurrency limit from env `INGESTION_CONCURRENCY` (default 20).

**Step 3: Write minimal implementation**
Update `main.go`.

---

### Task 6: Frontend API (List Pages)

**Files:**
- Modify: `apps/backend/features/source/handler.go`
- Test: `apps/backend/features/source/handler_test.go`

**Requirements:**
- **Endpoint:** `GET /sources/:id/pages`
- **Response:** JSON list of pages with status, depth, error.

**Step 1: Write failing test**
Test HTTP handler returns 200 and list of pages.

**Step 3: Write minimal implementation**
Add handler method `GetSourcePages`, wire to router.


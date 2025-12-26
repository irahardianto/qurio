# Key Learnings: Ingestion Testing & Weaviate Patterns
*Derived from the "Idempotency/Re-sync" investigation (Dec 2025)*

## 1. E2E Testing Strategies

### Data Isolation in Deduplicated Systems
**Problem:** The backend uses SHA256 content hashing to prevent duplicate uploads. E2E tests using hardcoded strings (even with different filenames) were silently rejected by the backend as duplicates, causing tests to fail or act unpredictably.
**Rule:** When testing ingestion pipelines, **always ensure file content is unique per test run**.
**Pattern:**
```typescript
const timestamp = Date.now();
// Bad: const content = "Fixed string";
// Good:
const content = `# Test Doc ${timestamp}\n\nUnique content for run ${timestamp}`;
```

### Robust Polling for Async UI
**Problem:** Background workers (ingestion) take variable time. The Frontend fetches data only once on mount. Tests asserting success immediately after an API call often hit stale UI states (0 chunks) because the worker hasn't finished or the UI hasn't refreshed.
**Rule:** Do not rely on transient states (like `in_progress`). **Poll the final state by reloading.**
**Pattern:**
```typescript
await expect(async () => {
    await page.reload(); // Force fetch fresh data
    await expect(chunkLocator.first()).toBeVisible({ timeout: 2000 });
}).toPass({ timeout: 60000 });
```

### Resource-Dependent Timeouts
**Problem:** PDF processing (OCR via `docling`) is CPU-intensive. Standard 30s timeouts fail in CI/Docker environments.
**Rule:** Set explicit, generous timeouts (300s+) for CPU-bound test steps.

## 2. Weaviate & Vector Database

### Tokenization & Exact Matching
**Observation:** There was a fear that `tokenization: "word"` (Weaviate default) would break exact filtering for UUID strings (e.g., `sourceId`).
**Learning:** Weaviate's `Equal` operator **successfully handles exact matches** for UUIDs and URL strings even with `word` tokenization. It does not require changing the schema to `field` tokenization for standard UUID filtering.

## 3. Backend Implementation

### Validation Error Visibility
**Observation:** The "Duplicate detected" error was returned by the backend but ignored/misinterpreted as a "silent failure" in the test.
**Rule:** Ensure backend validation errors (409 Conflict) are distinct from processing failures, and check logs for specific validation messages when tests fail mysteriously.

### Distributed Systems & Reset State
**Problem:** A "ReSync" operation merely reset the parent status to `in_progress`. However, the child tasks (pages) remained `completed` in the DB. The distributed worker's idempotency check (ON CONFLICT DO NOTHING) saw existing completed pages and refused to re-queue them, causing the system to hang immediately.
**Learning:** In stateful distributed systems (like a crawler frontier), a "Restart" must explicit **clean the state** (delete child records) to force the logic to re-evaluate and re-queue tasks.

### Worker Reliability & Timeouts
**Problem (StreamClosedError):** The NSQ client dropped connections during long processing tasks. This happened because the message "touch" (heartbeat) interval (30s) was too close to the server's timeout threshold, especially when the event loop was busy with heavy I/O.
**Fix:** Drastically reduce the touch interval (e.g., to 10s) to ensure heartbeats are sent reliably even under load.

**Problem (Crawl Timeout):** Large single-page documentation files (like `llms-full.txt`) frequently timed out with the default 120s limit.
**Fix:** Increase specific operation timeouts (to 300s+) for web crawling tasks to accommodate large payloads.

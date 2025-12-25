---
name: technical-constitution
description: Implementation plan for MVP Part 4.2 (Advanced Ingestion & Retrieval).
---

# Implementation Plan - MVP Part 4.2: Advanced Ingestion & Retrieval

**Ref:** `2025-12-25-qurio-mvp-part4-2`
**Feature:** Sitemap Support, llms.txt, Re-sync Integrity, Cohere Reranker
**Status:** Draft

## 1. Scope
Implement "Advanced Ingestion" features (Sitemaps, `llms.txt`) to improve crawl quality and "Advanced Retrieval" (Cohere Reranker) to complete the retrieval pipeline. Critically, fix the "Re-sync" data integrity issue where old chunks were not being deleted.

**Gap Analysis:**
- **Re-sync Integrity:** `ResultConsumer` currently appends new chunks without deleting old ones (duplicate data).
- **Ingestion:** `crawl4ai` integration lacks `sitemap.xml` and `llms.txt` discovery logic.
- **Reranking:** Cohere provider is missing in the backend adapter.

## 2. Requirements

### Functional
- **Re-sync:** When processing a page result, the system MUST delete all existing chunks for that `source_id + url` tuple before inserting new ones.
- **Sitemap:** The worker MUST detect `sitemap.xml` (if configured) and seed the crawler with those URLs.
- **llms.txt:** The worker MUST detect `llms.txt` at the root, parse it, and prioritize those URLs.
- **Cohere:** The backend MUST support `rerank_provider="cohere"` using `https://api.cohere.ai/v1/rerank`.

### Non-Functional
- **Performance:** `DeleteChunksByURL` must be efficient (batch delete in Weaviate).
- **Reliability:** Sitemap fetching should fail gracefully (fallback to recursive crawl).

## 3. Tasks

### Task 1: Vector Store Interface Update (DeleteChunks)
**Files:**
- Modify: `apps/backend/internal/worker/interfaces.go` (Create if missing or find definition)
- Modify: `apps/backend/internal/adapter/weaviate/store.go`
- Test: `apps/backend/internal/adapter/weaviate/store_test.go`

**Requirements:**
- Add `DeleteChunksByURL(ctx context.Context, sourceID string, url string) error` to `VectorStore` interface.
- Implement in Weaviate adapter using `batch.DeleteObjects`.

**Step 1: Write failing test**
Update `store_test.go` to insert chunks, call delete, and verify they are gone.

**Step 3: Implementation**
```go
// internal/adapter/weaviate/store.go
func (s *Store) DeleteChunksByURL(ctx context.Context, sourceID, url string) error {
    // Weaviate Batch Delete API
    // WHERE source_id = sourceID AND source_url = url
    return s.client.Batch().ObjectsBatcher().DeleteObjects(
        models.BatchDelete{
            Match: &models.BatchDeleteMatch{
                Class: "DocumentChunk",
                Where: &models.WhereFilter{
                    Operator: "And",
                    Operands: []*models.WhereFilter{
                        {Path: []string{"source_id"}, Operator: "Equal", ValueString: &sourceID},
                        {Path: []string{"source_url"}, Operator: "Equal", ValueString: &url},
                    },
                },
            },
        },
    )
}
```

### Task 2: Result Consumer Cleanup Logic
**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go`

**Requirements:**
- Before storing chunks, call `DeleteChunksByURL`.

**Step 1: Implementation**
```go
// HandleMessage
// ...
// 3. Delete Old Chunks (Idempotency)
if err := h.store.DeleteChunksByURL(ctx, payload.SourceID, payload.URL); err != nil {
    slog.Error("failed to delete old chunks", "error", err)
    return err // Retry on error to ensure consistency
}

// 4. Embed & Store
// ...
```

### Task 3: Backend Cohere Reranker
**Files:**
- Modify: `apps/backend/internal/adapter/reranker/client.go`

**Requirements:**
- Implement `rerankCohere` method.
- Endpoint: `https://api.cohere.ai/v1/rerank`.
- Model: `rerank-english-v3.0`.

**Step 1: Implementation**
```go
func (c *Client) rerankCohere(ctx context.Context, query string, docs []string) ([]int, error) {
    // Request body: { "model": "rerank-english-v3.0", "query": query, "documents": docs, "top_n": len(docs) }
    // Response: { "results": [{ "index": 0, "relevance_score": 0.9 }] }
    // ...
}
```

### Task 4: Worker Sitemap & llms.txt Support
**Files:**
- Modify: `apps/ingestion-worker/handlers/web.py`
- Modify: `apps/ingestion-worker/requirements.txt` (Ensure `crawl4ai` is up to date)

**Requirements:**
- **Sitemap:** Use `crawl4ai.AsyncUrlSeeder` with `source="sitemap"`.
- **llms.txt:** Manual fetch of `/llms.txt`. Parse links.
- **Priority:** `llms.txt` > `sitemap` > Recursive.

**Step 1: Implementation (web.py)**
```python
# Update handle_web_task
from crawl4ai import AsyncUrlSeeder, SeedingConfig

async def discover_urls(url: str) -> list[str]:
    urls = []
    # 1. Check llms.txt
    # ... fetch url/llms.txt ... parse ... append to urls ...
    
    # 2. Check Sitemap (using Seeder)
    async with AsyncUrlSeeder() as seeder:
        config = SeedingConfig(source="sitemap")
        sitemap_urls = await seeder.urls(url, config)
        urls.extend([u['url'] for u in sitemap_urls])
        
    return urls

# In handle_web_task:
# If max_depth > 0:
#   seed_urls = await discover_urls(url)
#   # Pass seed_urls to crawler config or queue
```

### Task 5: Integration Check
**Files:**
- Test: `apps/e2e/tests/ingestion.spec.ts`

**Requirements:**
- Verify that re-ingesting the same URL does not increase total chunk count (proof of cleanup).

**Step 1: Implementation**
```typescript
test('Re-ingestion replaces chunks', async ({ request }) => {
   // 1. Ingest URL
   // 2. Count chunks
   // 3. Re-ingest same URL
   // 4. Count chunks -> Should be same, not double
});
```

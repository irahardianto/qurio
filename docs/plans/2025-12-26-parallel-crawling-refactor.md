# Parallel Crawling & Distributed Ingestion Refactor

## 1. Objective
Transform the current "Monolithic Batch Crawl" architecture into a **Distributed, Page-Level Parallel** system. This ensures:
- **Scalability:** Multiple workers can process pages from the same website simultaneously.
- **Resilience:** A failure on one page does not discard the entire crawl.
- **Real-time Visibility:** The frontend can show exactly which pages are pending, processing, or completed.
- **Decoupled Embedding:** Embedding and Vectorization happen immediately per page, not after the entire site is crawled.

## 2. Architecture Comparison

### Current (Batch)
1. **User** submits URL.
2. **Worker** receives task.
3. **Worker** recursively crawls 100 pages (holding all in memory).
4. **Worker** finishes and sends list of 100 pages to Backend.
5. **Backend** processes all 100 pages.
6. **Result:** User waits minutes/hours with no feedback until 100% done.

### Proposed (Distributed Stream)
1. **User** submits URL.
2. **Backend** creates `Source` and **1** `SourcePage` (Seed).
3. **Backend** pushes **1** Job (Seed) to NSQ.
4. **Worker A** picks up Seed Job.
   - Crawls Seed.
   - Extracts Content + **Links**.
   - Sends Result to Backend.
5. **Backend** receives Seed Result.
   - Embeds & Stores Seed Content.
   - **Discovers** new links from result.
   - Creates `SourcePage` records for new links (Deduplication).
   - **Enqueues** new Jobs for new links.
6. **Workers A, B, C...** pick up new Jobs in parallel.
7. **Frontend** polls/streams `source_pages` status to show real-time progress bars.

## 3. Addressing Bottlenecks (Non-Blocking Flow)
A critical requirement is that the Backend's embedding process (which can be slow) must not block the discovery of new links.

**Scenario:** Worker A finishes Page 2. Worker B finishes Page 3.
**Risk:** If the Backend processes results sequentially, Page 3's links (Depth 2) would wait for Page 2 to finish embedding.

**Solution: Concurrent Result Handlers**
We will configure the Backend's NSQ Consumer to use `AddConcurrentHandlers`.
- This ensures that **Embedding** and **Link Discovery** for multiple pages happen in parallel threads.
- As soon as *any* page is processed, its children are immediately pushed to the queue, available for any idle Worker.
- **Result:** The crawl "fans out" exponentially as fast as workers can pick up tasks, limited only by the number of configured workers, not by the serialization of embedding.

## 4. Detailed Implementation Plan

### Phase 1: Database Schema
Create a new table `source_pages` to track the state of the "Crawl Frontier".

```sql
CREATE TABLE source_pages (
    id UUID PRIMARY KEY,
    source_id UUID REFERENCES sources(id),
    url TEXT NOT NULL,
    status TEXT DEFAULT 'pending', -- pending, processing, completed, failed
    depth INTEGER DEFAULT 0,
    error TEXT,
    UNIQUE(source_id, url)
);
```

### Phase 2: Ingestion Worker (Python) Refactor
Simplify the worker to be a "dumb" executor. It shouldn't know about recursion or depth limits, only about "Process this URL".
- **Modify `handlers/web.py`:**
  - Remove `BFSDeepCrawlStrategy` (recursion).
  - Change to single-page crawl logic.
  - Add **Link Extraction**: Extract all internal links from the crawled HTML/Markdown.
- **Output:** Return `{ "content": "...", "links": ["/about", "/docs"] }`.
- **Modify `main.py`:**
  - Pass the discovered `links` field in the NSQ payload to the backend.

### Phase 3: Backend (Go) Logic
The Backend becomes the "Coordinator".
- **Result Consumer (`worker/result_consumer.go`):**
  - **Process Content:** Chunk, Embed, Store (Existing logic).
  - **Process Links:**
    - If `current_depth < max_depth`:
      - Filter links (remove external domains, ignore existing `source_pages` for this source).
      - Insert new `source_pages` (Bulk Insert).
      - **Publish** new tasks to `ingest.task` topic.
  - **Update Status:**
    - Mark current `source_page` as `completed`.
    - Check if all pages for `source_id` are final. If so, mark `source` as `completed`.
- **Source Service (`features/source`):**
  - When creating a Source, insert the initial `source_pages` record for the seed URL.
- **Main Config (`main.go`):**
  - Configure `consumer.AddConcurrentHandlers(handler, 50)` (or configurable limit) to ensure high-throughput processing of incoming results.

### Phase 4: Frontend Visualization
- **New Endpoint:** `GET /sources/{id}/pages`
  - Returns list of pages with status.
- **UI:**
  - Replace simple spinner with a Progress Bar (e.g., "Processed 45/120 pages").
  - Show list of "Active Crawls".

## 5. Configuration & Concurrency
- **Worker Scaling:** You can now run `docker-compose up -d --scale ingestion-worker=5` to run 5 parallel workers.
- **Concurrency per Worker:** We will expose `NSQ_MAX_IN_FLIGHT` as an environment variable to control how many pages one single worker process handles concurrently (utilizing Python's `asyncio`).

## 6. Migration Strategy
1. **Apply DB Migration.**
2. **Deploy Backend** (to handle new message format with `links` and concurrency).
3. **Deploy Worker** (switched to single-page mode).
4. **Legacy Handling:** Old `pending` jobs might fail or behave oddly during the switch, but since this is a dev environment, we will assume a clean slate or manual retry is acceptable.

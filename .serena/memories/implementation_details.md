# Implementation Details

## Ingestion System (Distributed)
The ingestion system uses a distributed page-level crawl architecture.

### Components
1. **Database**:
   - `sources` table: Stores configuration (max_depth, exclusions).
   - `source_pages` table: Tracks individual pages (URL, status, depth).

2. **Ingestion Worker (Python)**:
   - Processes single pages.
   - Extracts content (markdown) and internal links.
   - Returns result to NSQ `ingest.result`.
   - Uses `Crawl4AI` with `AsyncWebCrawler`.

3. **Backend (Go)**:
   - **Producer**: Creates `Source` and seed `SourcePage`. Publishes seed task to `ingest.task`.
   - **Consumer**: Listens to `ingest.result`.
     - Processes content (chunking, embedding, vector storage).
     - **Link Discovery**: Filters new links, deduplicates against `source_pages`, creates new pages, and enqueues new tasks if `depth < max_depth`.
     - **Concurrency**: Uses `AddConcurrentHandlers` (configured by `INGESTION_CONCURRENCY`).

### Flow
1. User creates Source (Web).
2. Backend saves Source, creates Seed Page (Depth 0), publishes Task.
3. Worker picks up Task, crawls URL, extracts Links.
4. Worker publishes Result (Content + Links).
5. Backend Consumer processes Result.
   - Stores Chunks.
   - If `depth < max_depth`:
     - Filters links (internal only, exclusions).
     - Bulk inserts new `SourcePage` records (ignoring duplicates).
     - Publishes new Tasks for new pages (Depth + 1).
   - Updates Page Status to "completed".
6. Frontend polls `GET /sources/{id}/pages` to update the real-time progress bar and "Active Crawls" list.

## Search
- **Hybrid Search**: Combines BM25 (Keyword) and Vector Similarity.
- **Reranking**: Optional reranking step using Jina/Cohere.
- **Dynamic Tuning**: Alpha parameter (0.0 - 1.0) controls weight between keyword and vector search.

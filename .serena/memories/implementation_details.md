# Implementation Details

This document logs significant architectural decisions, feature implementations, and system refinements.

## 1. Distributed Ingestion Pipeline (Jan 2026)

### Architecture
The ingestion pipeline has been refactored into a distributed micro-pipeline to decouple coordination from heavy lifting.
- **ResultConsumer (Coordinator):** Consumes `ingest.result`, validates content, manages idempotency (deletes old chunks), splits content, and publishes to `ingest.embed`. It does *not* embed or store data.
- **EmbedderConsumer (Worker):** Consumes `ingest.embed`, reconstructs context, generates embeddings via `Embedder`, and stores vectors via `VectorStore`.
- **Scaling:** Scaling is managed via Docker Compose replicas (`INGESTION_WORKER_WEB_REPLICAS`, `INGESTION_WORKER_FILE_REPLICAS`, `BACKEND_WORKER_REPLICAS`).

### Parallelization
Processing is split into dedicated NSQ topics to prevent long-running tasks from blocking quick ones.
- `ingest.task.web`: Dedicated to web crawling.
- `ingest.task.file`: Dedicated to file processing.
- `ingest.result`: Shared topic for results.

### Reliability & Error Handling
- **Smart Retries:** Ingestion Worker implements "Smart Retry" for transient errors (Timeout, Connection) with configurable exponential backoff (`RETRY_MAX_ATTEMPTS`, `RETRY_INITIAL_DELAY_MS`, `RETRY_BACKOFF_MULTIPLIER`). Permanent errors fail immediately. Exposed via `.env` and `docker-compose.yml`.
- **Timeouts:** `CRAWLER_PAGE_TIMEOUT` is configurable (default 60s) to handle slow documentation sites.
- **Concurrency:** Internal worker concurrency (`WORKER_SEMAPHORE`) is aligned with `NSQ_MAX_IN_FLIGHT` to prevent overload.
- **Large Payloads:** NSQ server and clients are configured with a 10MB limit (`NSQ_MAX_MSG_SIZE`) to support large PDF markdown generation.

## 2. Source Management

### Source Naming
- **Mandatory Naming:** All sources (Web and File) require a `name` at creation.
- **Persistence:** Names are stored in Postgres and propagated through the pipeline (via `SourceFetcher`) to be attached to chunks for context.
- **API:** `POST /sources` and `POST /sources/upload` require the `name` field.

### Pagination & Polling
- **Backend:** `ChunkStore` and `VectorStore` support `limit`/`offset` pagination. API `GET /sources/{id}` accepts `limit`, `offset`, and `exclude_chunks`.
- **Frontend:** Implements "Load More" functionality and uses lightweight polling (metadata only) to update status without resetting scroll position.

## 3. MCP Integration

- **Tool Enhancements:** `qurio_list_sources` returns the `url` field for sources, populated from the database, to provide better context to the LLM.

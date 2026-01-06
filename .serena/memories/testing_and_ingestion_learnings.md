# Testing and Ingestion Learnings (Updated 2026-01-07)

## Ingestion Worker
- **Architecture**: Hybrid async/sync model. Web crawling is async (crawl4ai), file processing is sync (docling) offloaded to `pebble.ProcessPool`.
- **Testing**: Heavy reliance on mocking (pebble, crawl4ai).
- **Recent Improvements (2026-01-06)**:
    - **Metadata Extraction**: Logic extracted to pure functions (`extract_metadata_from_doc`). Handled edge cases (callables, NoneTypes) defensively.
    - **Zombie Tasks**: `touch_loop` now uses `asyncio.wait_for(event.wait())` for immediate exit, preventing zombie processes on cancellation.
    - **Concurrency**: Global `WORKER_SEMAPHORE` (8) enforced in `main.py` for all task types.
    - **Error Handling**: `correlation_id` added to all NSQ failure payloads.
    - **ResultConsumer Hardening**: Adopted "Poison Pill" testing strategy for handling malformed JSON messages without crashing. Explicit testing for embedding service timeouts and failures.

## Testing Strategy Updates
- **Metadata**: Use `pytest.mark.parametrize` for table-driven testing of extraction logic.
- **Concurrency**: explicit semaphore saturation tests required.
- **Logging**: Must verify stdlib bridge to structlog.
- **Backend Test Patterns**:
    - **Dependency Injection**: Enforce interface-based mocks for `Database`, `VectorStore`, and `TaskPublisher` in `apps/backend/internal/app`.
    - **MCP Handlers**: Use comprehensive table-driven tests covering all tools and negative paths (MethodNotFound, InvalidParams).
    - **Adapters**: Simulate network errors (503, GraphQL errors) using `httptest` for Weaviate and dynamic key rotation checks for Gemini.

## Backend Integration Testing (2026-01-07)
- **Infrastructure**: Introduced `internal/testutils/IntegrationSuite` using `testcontainers-go`.
- **Containers**: Real ephemeral instances of:
    - **Postgres (16-alpine)**: Verified with `golang-migrate`.
    - **Weaviate (latest)**: Verified with generic container + REST API wait strategy.
    - **NSQ (v1.3.0)**: Verified basic producer connectivity.
- **Coverage**:
    - **Source Repo**: Full CRUD, Deduplication, Page Management against real DB.
    - **Weaviate Store**: Full CRUD, Hybrid Search (requires vector input for 'none' vectorizer), Metadata filtering.
    - **Worker Flow**: `ResultConsumer` tested end-to-end (Message -> DB -> Weaviate) with mocked Embedder.
    - **MCP Handlers**: Integration test validates `qurio_search` and `qurio_read_page` against real Weaviate/DB populated with seed data.

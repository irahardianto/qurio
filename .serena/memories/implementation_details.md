# Implementation Details

## Backend Structure
- **Core:** `apps/backend/internal/app/app.go` wires everything.
- **MCP:** `apps/backend/features/mcp/handler.go` implements Model Context Protocol.
    - **Transport:** Stateless HTTP POST (Single Request/Response) via `/mcp`. (Refactored Jan 2026).
    - **Tools:** `qurio_search` (Hybrid), `qurio_list_sources`, `qurio_list_pages`, `qurio_read_page`.
- **Ingestion:** `apps/ingestion-worker` (Python) handles crawling/parsing.
- **Retrieval:** `apps/backend/internal/retrieval` handles Weaviate/Rerank logic.

## Testing Patterns
- **Unit:** Co-located `_test.go`. Use `httptest` for handlers.
- **Integration:** `_integration_test.go` in same package.
- **E2E:** `apps/e2e` (Playwright).

## Configuration
- **Hybrid:** `config.yaml` + Environment Variables.
- **Secrets:** Injected via ENV.

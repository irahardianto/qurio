# Implementation Details

## Configuration & Settings
- **Settings Table:** Stores global configuration (singleton row `id=1`).
    - `rerank_provider` (text): 'none', 'jina', 'cohere'.
    - `rerank_api_key` (text).
    - `gemini_api_key` (text).
    - `search_alpha` (float): Global default for hybrid search balance (0.0=Keyword, 1.0=Vector).
    - `search_top_k` (int): Global default for max results (UI label: "Max Results").

## Search & Retrieval
- **Hybrid Search:** Uses Weaviate's `alpha` parameter.
- **Smart Tooling:** The `search` MCP tool accepts optional `alpha` and `limit` arguments.
    - If provided by agent, these override the global defaults.
    - If missing, global defaults are used.
- **Agent Guide:** The `search` tool description includes a table guiding the agent on when to use specific alpha values (e.g., "0.0 for Error Codes").

## Ingestion Architecture
- **Worker (Python):** Handles crawling (`crawl4ai`) and file conversion (`docling`).
    - **Advanced Ingestion:** Supports `sitemap.xml` and `llms.txt` discovery to prioritize and seed URLs.
    - Consumes: `ingest.task` (NSQ)
    - Produces: `ingest.result` (NSQ) -> Backend
- **Backend (Go):**
    - Consumes: `ingest.result`
    - **Idempotency:** Calls `DeleteChunksByURL` before storing new chunks for a given source + URL to prevent duplicates during re-sync.
    - Actions: Chunking -> Embedding (Gemini) -> Storage (Weaviate).

## Frontend Architecture
- **Settings UI:**
    - "Search Balance" slider (controls `search_alpha`).
    - "Max Results" input (controls `search_top_k`).
    - Includes tooltips for user education.

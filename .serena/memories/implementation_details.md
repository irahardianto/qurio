# Implementation Details

## Backend Architecture (Go)
The backend follows a **Feature-Based Architecture** (`apps/backend/features/`), grouping logic by domain rather than technical layer.

### Core Features
- **Source (`features/source`)**: Manages ingestion sources (Web/File).
  - Uses `PostgresRepo` for metadata (including user-defined `Name`) and state (`source_pages` table).
  - Publishes tasks to NSQ (`ingest.task`).
  - Handles page-level status tracking (Pending -> Processing -> Completed/Failed).
  - **File Uploads**: Enforces 50MB limit at Nginx (client_max_body_size) and Backend (MaxBytesReader).
- **Job (`features/job`)**: Manages failed ingestion tasks (DLQ).
  - **Dead Letter Queue**: Failed worker tasks are saved to `failed_jobs` table via `JobRepository`.
  - **Error Handling**: Worker captures `original_payload` on failure, allowing exact retries.
  - **Retry Mechanism**: `POST /jobs/{id}/retry` re-publishes the `original_payload` to NSQ.
- **MCP (`features/mcp`)**: Implements Model Context Protocol.
  - **Transport**: Supports both SSE (`/mcp/sse`) and JSON-RPC (`/mcp/messages`).
  - **Context Propagation**: SSE messages use `context.WithoutCancel` to detach from request context, ensuring long-running tool executions persist despite client disconnection.
  - **Tools**:
    - `qurio_search`: Hybrid search with metadata filtering (type, language). Output formatted with Title/Type headers.
    - `qurio_fetch_page`: Retrieves full page content by URL, preserving code blocks.

- **Retrieval (`internal/retrieval`)**:
  - **Chunker**: Markdown-aware state machine with hierarchical prose splitting (Headers->Paragraphs->Lines) and strict Code Block preservation (regex-based) and API endpoint detection.
  - **Store (Weaviate)**: Implements hybrid search with dynamic filter builder. Stores rich metadata (`Title`, `Type`, `Language`, `SourceName`) and `Contextual Embeddings` (Title+Source+Path+URL+Type prepended to vector).
  - **ResultConsumer**: Normalizes URLs (strips fragments) to prevent redundant crawls.

### Reliability & Maintenance
- **Janitor Service**: Background process (`ResetStuckPages`) runs every 5 minutes to recover pages stuck in `processing` state.
- **Observability**: `slog` is used for structured logging across the service.

### Ingestion Worker (Python)
The worker is a distributed consumer built with `pynsq`, `asyncio`, and `crawl4ai`.

- **Reliability**:
  - **Robust Touch Loop**: Keeps NSQ connection alive.
  - **Discovery**: Excluded tags removed to allow Sidebar/Nav link discovery.
  - **Error Reporting**: Captures `original_payload` on failure.
- **Handlers**:
  - `web.py`: Uses `AsyncWebCrawler` (Chromium) + `LLMContentFilter` (Gemini Flash). Extracts page `title` via regex/metadata fallback and `path` (breadcrumbs) from URL.
  - `file.py`: Uses `docling` for local file conversion. Extracts title from filename, returns `path` metadata.
- **Pipeline**: Propagates `title` and `path` in NSQ payload to Backend.

## Frontend Architecture & Design System
The frontend (Vue 3) implements a custom "Sage" aesthetic.

### Feature Implementation
- **Source Details**: Displays rich metadata (Type, Language badges, Title) for ingested chunks.
- **Dashboard**: Real-time system stats.
- **Jobs Management**: Monitor and retry failed tasks.

## Data Flow
1. **Ingestion**: User -> Backend -> NSQ (`ingest.task`) -> Worker (Crawl + Title Extract) -> NSQ (`ingest.result` w/ Title) -> Backend (Consumer).
2. **Processing**: Consumer -> Chunker (Split Prose/Code) -> Embedder (Contextual String) -> Weaviate (Store Metadata + Vector).
3. **Retrieval**: MCP/Frontend -> Backend -> Weaviate (Filter + Hybrid Search) -> Result (w/ Metadata).

## API Standards
- **Chunk Serialization**: The Chunk API uses `snake_case` for all fields (e.g., `chunk_index`, `source_name`) to align with Python/JSON conventions.

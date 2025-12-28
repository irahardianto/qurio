# Implementation Details

## Backend Architecture (Go)
The backend follows a **Feature-Based Architecture** (`apps/backend/features/`), grouping logic by domain rather than technical layer.

### Core Features
- **Source (`features/source`)**: Manages ingestion sources (Web/File).
  - Uses `PostgresRepo` for metadata and state (`source_pages` table).
  - Publishes tasks to NSQ (`ingest.task`).
  - Handles page-level status tracking (Pending -> Processing -> Completed/Failed).
- **Job (`features/job`)**: Manages failed ingestion tasks (DLQ).
  - **Dead Letter Queue**: Failed worker tasks are saved to `failed_jobs` table via `JobRepository`.
  - **Error Handling**: Worker captures `original_payload` on failure, allowing exact retries.
  - **Retry Mechanism**: `POST /jobs/{id}/retry` re-publishes the `original_payload` to NSQ.
- **MCP (`features/mcp`)**: Implements Model Context Protocol.
  - **Transport**: Supports both SSE (`/mcp/sse`) and JSON-RPC (`/mcp/messages`).
  - **Context Propagation**: SSE messages use `context.WithoutCancel` to detach from request context, ensuring long-running tool executions persist despite client disconnection.
  - Integrates with `retrieval` service for RAG.

### Reliability & Maintenance
- **Janitor Service**: Background process (`ResetStuckPages`) runs every 5 minutes to recover pages stuck in `processing` state (e.g., due to worker crashes/OOM), resetting them to `pending` or `failed`.
- **Observability**: `slog` is used for structured logging across the service, with specific focus on retry mechanisms and worker interactions.

### Ingestion Worker (Python)
The worker is a distributed consumer built with `pynsq`, `asyncio`, and `crawl4ai`.

- **Reliability**:
  - **Robust Touch Loop**: Runs in background to keep NSQ connection alive. Cancels main task if connection drops (`StreamClosedError`).
  - **Robots.txt**: Enforced via `crawl4ai` config (`check_robots_txt=True`) to ensure politeness and avoid bans.
  - **Error Reporting**: Captures `original_payload` on failure and sends to `ingest.result` with `status: failed`.
- **Observability**:
  - **Structlog**: Configured to capture standard library logs (e.g. from `tornado`, `httpx`), ensuring consistent JSON logging format in production.
- **Handlers**:
  - `web.py`: Uses `AsyncWebCrawler` (Chromium) + `LLMContentFilter` (Gemini Flash) for content extraction. Returns standardized `list[dict]` structure.
  - `file.py`: Uses `docling` for local file conversion (PDF/Docx).

## Frontend Architecture & Design System
The frontend (Vue 3) implements a custom "Sage" aesthetic: technical, precise, and grounded.

### "Sage" Design System
- **Theme**: Void Black (`#0F172A`) and Cognitive Blue (`#3B82F6`) using HSL variables for dynamic theming.
- **Typography**: `Inter` for UI elements and `JetBrains Mono` for data/code presentation.
- **Visuals**: Glassmorphism (`backdrop-blur`), sharp borders (`border-slate-800`), and subtle glow effects on interactions.

### Component & Layout Patterns
- **Full-Width Layout**: All views utilize the full viewport width (`w-full`) with responsive padding (`p-6 lg:p-10`).
- **Master-Detail Views**: Used for complex data like Ingested Chunks (`SourceDetailView`).
- **Hero Inputs**: Key actions (like "Add Source") use prominent, hero-style containers.
- **Interactive Tooltips**: Settings and help text use rich, animated tooltips.

### Feature Implementation
- **Dashboard**: Real-time system stats displayed in semi-transparent glass cards. Includes "Failed Jobs" widget.
- **Jobs Management**:
  - **Monitor**: Dedicated `/jobs` view listing failed ingestion tasks.
  - **Retry**: UI action to trigger backend retry for failed jobs.
  - **Dev Proxy**: Vite configured to proxy `/api` to backend during development.
- **Sources Library**: Grid-based card layout for managing active ingestion targets.

## Data Flow
1. **Ingestion**: User -> Backend (Create Source) -> NSQ (`ingest.task`) -> Worker (Crawl) -> NSQ (`ingest.result`) -> Backend (ResultConsumer).
2. **Failure**: Worker (Error) -> NSQ (`ingest.result` w/ Error & Original Payload) -> Backend -> `failed_jobs` table.
3. **Retry**: User -> Backend (Retry Endpoint) -> NSQ (`ingest.task` w/ Original Payload).

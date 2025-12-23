# Implementation Details

## Technical Compliance & Stabilization (Part 3.5)

### Backend (Go)
- **Error Handling:** Standardized JSON error envelopes (`{"status": "error", "error": {...}, "correlationId": "..."}`) for all HTTP responses.
- **Correlation IDs:** Generated at ingress (MCP), passed via `X-Correlation-ID`, and included in all structured logs.
- **Status Lifecycle:** 
  - `in_progress`: Set immediately by Service before publishing to NSQ.
  - `completed`: Set by Result Consumer upon successful processing.
  - `failed`: Set by Result Consumer upon receiving failure payload from worker.
- **Logging:** `log/slog` with structured context.

### Ingestion Worker (Python)
- **Logging:** Migrated to `structlog`. JSON in production, Console in dev.
- **Reliability:**
  - **Dynamic Timeouts:** Recursive crawls use `timeout = 60s + (60s * depth)` to accommodate LLM processing.
  - **Failure Handling:** Catches exceptions/timeouts, publishes `status="failed"` result, and **acks** the message to prevent infinite retries.
  - **Deep Crawling:** Uses `crawl4ai`'s `BFSDeepCrawlStrategy` with correctly initialized `FilterChain` (empty list if no exclusions).
- **Configuration:**
  - Removed redundant `GEMINI_API_KEY` env var (uses dynamic injection).
  - Explicitly sets `temperature=1.0` for Gemini 3 models (though LiteLLM may still warn).

### Infrastructure
- **Docker:** Worker container requires rebuild on code changes (`build: .` without volume mount for code).
- **Storage:** Named volume `qurio_uploads` shared between `backend` (`/var/lib/qurio/uploads`) and `ingestion-worker` (`/var/lib/qurio/uploads`) for atomic file processing.

### Document Upload (Part 3.6) - COMPLETED
- **Infrastructure:**
    - Replaced `/tmp/qurio-uploads` with named volume `qurio_uploads` in `docker-compose.yml`.
- **Backend:**
    - **Migration:** `000007_add_source_type.up.sql` added `type` column to `sources` table.
    - **Endpoint:** `POST /sources/upload` handles `multipart/form-data`, saves to `/var/lib/qurio/uploads`, and creates source with `type="file"`.
    - **ReSync:** Updated to send `path` payload for file sources.
- **Frontend:**
    - **UI:** Tabbed `SourceForm` for Web/File modes.
    - **UX:** `SourceList` and `SourceDetail` display filenames (without UUID prefix) and File Icon for file sources.

## Reliability & Standardization (Part 3.7)

### Backend (Go)
- **Middleware:** Implemented `CorrelationID` middleware (`apps/backend/internal/middleware`) to ensure every request has a unique `X-Correlation-ID`.
- **API Response:** Standardized success responses to use `{ "data": ..., "meta": ... }` envelope, matching the error format.
- **Worker Timeouts:** Enforced 60s hard timeout on Embed and StoreChunk operations in `ResultConsumer`.

### Ingestion Worker (Python)
- **Handler Contract:** Standardized handlers (`file.py`, `web.py`) to return `list[dict]` to simplify the main loop.
- **File Handling:** Normalized file ingestion to return a single-item list with `url` (path) and `content`.

### Frontend
- **Components:** Added `Textarea` UI component matching `shadcn` styling.
- **Forms:** Updated `SourceForm` to use the standardized `Textarea` for configuration.
- **State Management:** Updated `source.store.ts` and `settings.store.ts` to unwrap the standardized API response envelope (`res.json().data`).


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

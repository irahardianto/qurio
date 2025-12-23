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

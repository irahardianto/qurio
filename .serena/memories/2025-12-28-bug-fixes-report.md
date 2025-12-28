# Bug Fixes Report - 2025-12-28

## Context Propagation
- **Fixed:** MCP SSE messages now use `context.WithoutCancel` to detach from the request context, ensuring long-running tool executions are not interrupted by client disconnection.
- **Verification:** Unit test `TestHandleMessage_ContextPropagation` confirms async context usage.

## Ingestion Reliability
- **Standardized:** Web ingestion handler now returns a `list[dict]` consistent with file ingestion.
- **Janitor:** Implemented `ResetStuckPages` background job (5 min ticker) to recover stuck processing pages.
- **Verification:** `test_handle_web_task` passes with list assertion. `TestService_ResetStuckPages` passes.

## Observability
- **Job Service:** Added `slog` logging to `Retry` mechanism for better traceability.
- **Worker:** Configured `structlog` to capture standard library logs (e.g. from `tornado`, `httpx`), ensuring consistent JSON logging in production.
- **Verification:** Worker tests confirm JSON log formatting for stdlib calls.

## Tests
- Added `apps/backend/features/mcp/handler_test.go`
- Added `apps/ingestion-worker/tests/test_logger.py`
- Updated `apps/backend/features/source/source_test.go` and `handler_test.go`
- Updated `apps/ingestion-worker/tests/test_handlers.py` and `test_nsq.py`
- Fixed mocks in `result_consumer_test.go`

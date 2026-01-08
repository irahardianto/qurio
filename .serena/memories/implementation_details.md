## Bug Fixes & Inconsistencies (2026-01-08)
- **Backend Ingestion**: Fixed trace context loss in `ResultConsumer`. Correlation IDs are now extracted/generated before context initialization, ensuring all logs are traceable.
- **Worker Logging**: Enforced JSON formatting for `tornado` and `nsq` loggers in production to prevent "split-brain" logging formats.
- **Retrieval**: `SearchResult` now exposes `SourceName` at the top level, mapped from Weaviate metadata.
- **Testing**: Added unit tests for `MCP Handler` error paths, `Job.Count`, and `Reranker.DynamicClient` glue logic.
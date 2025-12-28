# Project Status Update - MVP Complete

The Qurio MVP is now functionally complete.

**Key Achievements:**
- **Infrastructure:** Docker Compose stack (Postgres, Weaviate, NSQ, Go Backend, Python Worker, Vue Frontend) verified.
- **Core Features:**
  - **Distributed Web Ingestion** (Page-level parallel crawling).
  - Web & File Ingestion (with deduplication).
  - Hybrid Search (BM25 + Vector) with optional Reranking (Jina/Cohere).
  - MCP Endpoint (JSON-RPC 2.0 & SSE).
  - Admin Dashboard (Sources, Stats, Failed Jobs).
- **Quality Assurance:**
  - Backend integration tests passing.
  - End-to-end (Playwright) tests passing.
  - API standardized (JSON envelopes, correlation IDs).
  - Structured logging (slog/structlog) implemented end-to-end.
  - Resilience features (Timeouts, DLQ/Retry) verified.
  - Known Issue: Background janitor for stuck jobs pending implementation.

**Latest Version:** v0.2.0-MVP (as per Sidebar)
**Documentation:** Updated README.md with setup and usage instructions.

**Ready for:**
- Beta testing / User acceptance testing.
- Deployment.

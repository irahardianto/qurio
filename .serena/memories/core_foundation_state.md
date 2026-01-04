# Core Foundation State (Jan 4, 2026)

## Ingestion Pipeline Stability
- **Architecture:** Robust multi-process worker using `pebble`.
- **Reliability:**
  - Hard timeouts (30 mins) with auto-kill for stuck processes.
  - Resource limits (8 CPUs, 8GB RAM) enforced via Docker.
  - Thread safety enforced via environment variables (`OMP_NUM_THREADS=2`).
- **Performance:**
  - Worker: 8 concurrent workers processing files.
  - Backend: Parallel chunk embedding (5 concurrent routines).
- **Capabilities:**
  - Handles large books (e.g., 50MB, 500+ pages).
  - Extracts rich metadata (Author, Created Date, Page Count).
  - Standardized on Docling v2 API.

## Known Limitations
- **Progress Reporting:** "Pending" state is a black box. No page-level progress. (Proposal created: `docs/2026-01-04-proposal-worker-realtime-progression.md`)
- **Search API:** Metadata stored but not yet returned to frontend. (Proposal created: `docs/2026-01-04-proposal-search-metadata-returns.md`)

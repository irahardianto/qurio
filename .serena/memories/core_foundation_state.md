# Core Foundation State

## Architecture
- **Language:** Go 1.25+
- **Database:** PostgreSQL + pgvector
- **Orchestration:** Docker SDK for Go (Sidecars)
- **API:** Chi Router + Standard Lib
- **Agent Protocol:** MCP (Model Context Protocol) via SSE
- **LLM Provider:** Google Gemini (via `generative-ai-go`)

## Implemented Components
1.  **Project Management:**
    - Registration with git/stack detection.
    - Gate parsing (.monarch/gates.yaml).
    - Database schema for projects/tasks.

2.  **Runner Engine:**
    - `Executor`: Docker Exec wrapper for running commands.
    - `Manager`: Warm container lifecycle (Start, Get, Idle Timeout).
    - `Reaper`: Cleanup of orphaned containers on startup.
    - `Parsers`: Fail-closed parsing for `go test` and `eslint`.
    - `Eval Engine`: LLM-based evaluation for Snapshot and Diff analysis.
    - `RunnerService`: Orchestrates Standard (Docker) and LLM gates.

3.  **Agent Interface (MCP):**
    - `MCPServer`: Core server instance using `go-sdk`.
    - `SSEHandler`: Transport layer bridging HTTP to MCP (via `go-sdk`).
    - `Planner`: Tools for querying state (`list_projects`, `search_past_tasks`).
    - `Builder`: Tools for executing tasks (`claim_task`, `submit_attempt`) with Circuit Breaker.
    - `Qurio`: Knowledge tools (`qurio_search`, `qurio_fetch_page`) for documentation retrieval.

4.  **Utilities:**
    - `Crypto`: AES-GCM encryption for sensitive data.
    - `Settings`: Database-backed, encrypted settings store.

5.  **Ingestion & Retrieval (Qurio):**
    - **Chunking:** Markdown-aware splitting with strict code block preservation and **API endpoint detection**.
    - **Crawling:** URL normalization, dynamic sidebar link discovery, and **Breadcrumb (`path`) extraction**.
    - **Storage:** Weaviate schema with dynamic properties. **Contextual Embeddings** enriched with `Source Name`, `Path`, `Title`, `Type`, `URL`, `Author`, and `Created At`.
    - **Worker:** Python worker with `pynsq` and `docling` for file metadata extraction.

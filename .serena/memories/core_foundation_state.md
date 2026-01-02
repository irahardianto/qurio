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

4.  **Utilities:**
    - `Crypto`: AES-GCM encryption for sensitive data.
    - `Settings`: Database-backed, encrypted settings store.

5.  **Ingestion & Retrieval (Qurio):**
    - **Chunking:** Markdown-aware hierarchical splitting (Headers -> Paragraphs -> Lines) with strict code block preservation.
    - **Crawling:** URL normalization (fragment stripping) and dynamic sidebar link discovery.
    - **Storage:** Weaviate schema with dynamic property migration (`Title`, `Type`, `Language`) and Contextual Embeddings.

# Monarch - Project Overview

**Monarch** is an open-source, AI-native task management platform designed to prevent "architectural drift" in agentic workflows. It acts as an **Execution Supervisor**, utilizing the Supervisor Pattern to validate agent work against defined Non-Functional Requirement (NFR) Gates.

## Core Value Proposition
Monarch wraps task completion in strict, executable gates. It runs locally alongside the developer's code and uses **Docker sidecars** to validate code (Security, Performance, Testing) before allowing task closure.

## Key Features
*   **Supervisor Pattern:** Acts as a bridge between humans (Requirements) and Agents (Execution).
*   **NFR Gates:** Ephemeral Docker containers validate code (Tier A/B/C gates).
*   **Project Registration:** Auto-detects project stack (Go, Node, Python) and configures gates.
*   **Protocol Translation:** Converts raw tool logs into agent-readable "Unified Error Objects".
*   **Circuit Breaking:** Detects and blocks infinite loops in agent behavior (default 5 attempts).
*   **MCP Integration:** Exposes "Planner" and "Builder" toolsets via Model Context Protocol.
*   **LLM Evaluation:** AI-powered code review gates using Gemini Pro.

## Architecture
*   **Type:** Local-first, self-hosted platform.
*   **Components:** Single Go binary (API, MCP Server, State) located in `apps/backend`, PostgreSQL + pgvector, Docker SDK for orchestration.
*   **Interface:** Vue.js + Shadcn Dashboard located in `apps/frontend`.

## Current Status (2026-01-03)
*   Backend Core implemented (DB, Runner, API, Gates, Project).
*   **Execution Engine:** Universal Docker Executor and Tool Output Parsers (Go Test, ESLint) implemented.
*   **LLM Eval Engine:** Implemented with Gemini Pro integration, Snapshot/Diff analysis.
*   **MCP Server:** Implemented with SSE transport, `qurio_search` (hybrid+filter+source_id), `qurio_list_sources`, `qurio_list_pages`, and `qurio_fetch_page`.
*   **Ingestion (Qurio):** Advanced pipeline with API detection, breadcrumb extraction, and contextual embeddings.
*   Frontend pending.
*   **Planning:** Epic 5 (LLM Eval Engine) implementation complete.

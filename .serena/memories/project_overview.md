# Qurio - Project Overview

**Qurio** is an open-source, local-first **knowledge engine** designed to serve as a curated, grounded context provider for AI agents. It addresses the problem of hallucinations and context fragmentation in AI coding workflows by allowing developers to ingest, index, and retrieve high-quality, team-selected documentation directly from their local environment.

## Core Value Proposition
*   **Privacy First:** Runs locally via Docker, ensuring proprietary documentation stays within your network.
*   **Precision & Grounding:** Prioritizes retrieval of "ground truth" to prevent AI hallucinations.
*   **Developer-Centric:** Built for software engineering data, distinguishing between code, prose, APIs, and configuration.
*   **Standardized Access:** Exposes knowledge via the **Model Context Protocol (MCP)**, making it universally accessible to any MCP-compliant agent (Claude, Cursor, etc.).

## Key Features
*   **Specialized Ingestion:** Advanced pipeline (Python/Docling) that understands Markdown structure, preserving code block integrity and extracting metadata (language, title).
*   **Contextual Embeddings:** Injects document-level context (breadcrumbs, source) into vector embeddings to eliminate ambiguity in isolated chunks.
*   **Metadata-Based Filtering:** Allows agents to explicitly query for specific content types (e.g., "just the code," "API specs only") via `qurio_search`.
*   **Full Document Retrieval:** "Deep dive" capability (`qurio_fetch_page`) allows agents to read entire documents when search snippets are insufficient.
*   **Hybrid Search:** Combines keyword (BM25) and semantic (Vector) search with dynamic alpha tuning.

## Architecture
*   **Type:** Local-first, self-hosted platform.
*   **Backend:** Go (Standard Library) - Handles API, MCP Server, and State Management.
*   **Ingestion Worker:** Python - Uses Crawl4AI and Docling for high-fidelity web crawling and document parsing.
*   **Frontend:** Vue.js 3 + Shadcn/Tailwind - Dashboard for managing sources, viewing stats, and handling failed jobs.
*   **Storage:**
    *   **Weaviate:** Vector database for semantic search and chunk storage.
    *   **PostgreSQL:** Relational database for source metadata and job tracking.
    *   **NSQ:** Message queue for reliable async communication between Backend and Worker.

## Current Status (Jan 2026)
*   **Backend:** Core services (Source, Job, Retrieval) and MCP server implemented.
*   **Ingestion:** Robust pipeline supports web crawling, PDF processing, and automatic content type detection (Prose vs. Code).
*   **MCP Tools:**
    *   `qurio_search`: Search & Exploration tool (Hybrid: Keyword + Vector). Supports filtering by `type`, `language`, and `source_id`.
    *   `qurio_list_sources`: Discovery tool. Lists all available documentation sets.
    *   `qurio_list_pages`: Navigation tool. Lists pages within a source.
    *   `qurio_read_page`: Deep Reading tool. Retrieves full document content by URL.
*   **Frontend:** Dashboard implemented with Statistics and Failed Jobs management.
*   **Reliability:** Idempotent ingestion, retry mechanisms for failed jobs, and orphan chunk cleanup.
*   **Scalability:** Distributed Micro-Pipeline with decoupled coordination and embedding workers. Configurable concurrency and horizontal scaling (replicas) for all worker types.
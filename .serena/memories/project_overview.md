# Project Overview

## Mission
**Qurio** is an autonomous "Knowledge Engine" that ingests, processes, and serves high-quality technical knowledge. It combines a robust ingestion pipeline with a precise RAG (Retrieval-Augmented Generation) system, designed to power AI agents and technical assistants.

## Core Capabilities
- **Universal Ingestion**: Crawls websites (with Javascript support) and processes documents (PDF, Markdown).
- **Smart Chunking**: Uses LLM-powered content filtering and semantic chunking strategies.
- **Hybrid Search**: Combines Keyword (BM25) and Vector (Embeddings) search with dynamic alpha tuning.
- **MCP Integration**: Exposes knowledge via the Model Context Protocol for seamless agent integration.
- **Resilient Architecture**: Features a persistent Dead Letter Queue (DLQ) and retry mechanisms for ingestion reliability.

## Current State (v0.2.0-MVP)
- **Backend**: Fully functional Go service with NSQ messaging and Weaviate integration.
- **Worker**: Python-based distributed worker using `crawl4ai` and `docling`.
- **Frontend**: A polished, developer-centric "Sage" interface (Vue 3) featuring:
    - **Void Black** / **Cognitive Blue** aesthetic.
    - **Full-width responsive layouts** with glassmorphic cards.
    - **Master-Detail views** for inspecting ingested content.
    - **Drag-and-drop** file uploads and real-time job monitoring.

## Key Links
- **API Documentation**: See `api_endpoints.md`
- **Technical Stack**: See `tech_stack.md`
- **Architecture**: See `implementation_details.md`

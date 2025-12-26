# Project Overview

Qurio is a self-hosted, open-source ingestion and retrieval engine designed to provide grounded context for AI agents.

## Core Features
- **Ingestion:** Supports web crawling (with Sitemap/llms.txt support) and file uploads (PDF/DOCX via Docling).
- **Retrieval:** Hybrid search (Weaviate) with configurable Reranking (Jina/Cohere).
- **Interface:** MCP Protocol for agents, Admin UI for management.
- **Architecture:** Go Backend, Vue Frontend, Python Worker, PostgreSQL + Weaviate.

## Current Status
- **Date:** 2025-12-26
- **Completed:** 
  - Deployment (Docker)
  - Core Ingestion (Web/File) & Retrieval (MCP)
  - Settings (Alpha/TopK)
  - Bug Fixes & Standardization (Correlation IDs, API Envelopes)
  - Part 4.2: Advanced Ingestion (Sitemap/llms.txt), Re-sync Integrity, Cohere Reranker.
- **Next:** 
  - Part 5.1: Admin Completeness (Dashboard, Failed Jobs/DLQ), Source Cleanup, Documentation.
  - Final E2E Testing.

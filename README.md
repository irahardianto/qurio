# Qurio

Qurio is a self-hosted, open-source context retrieval engine designed to provide grounded knowledge to AI agents. It bridges the gap between your data (websites, files) and your LLM agents via the Model Context Protocol (MCP).

## Features

- **Ingestion:**
  - **Web Crawling:** Recursively crawls websites, respecting `robots.txt` and `sitemap.xml`.
  - **File Uploads:** Supports PDF, DOCX, and other formats via [Docling](https://github.com/DS4SD/docling).
  - **LLM-First Discovery:** Prioritizes `llms.txt` if available for cleaner context.
- **Retrieval:**
  - **Hybrid Search:** Combines keyword search (BM25) with vector search (Weaviate).
  - **Reranking:** Optimizes search results using Cohere or Jina AI rerankers.
  - **MCP Interface:** Standardized protocol for AI agents (like Claude Desktop) to query your knowledge base.
- **Management:**
  - **Dashboard:** Monitor sources, document counts, and failed jobs.
  - **Source Control:** Add, remove, re-sync, and configure sources (max depth, exclusions).
  - **Failed Job Handling:** View and retry failed ingestion tasks.

## Getting Started

### Prerequisites

- **Docker** and **Docker Compose** installed.
- **Git** installed.
- **Gemini API Key** (for embeddings and optional Reranking via Jina/Cohere).

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourusername/qurio.git
   cd qurio
   ```

2. **Configure Environment:**
   Copy `.env.example` to `.env` and fill in your secrets.
   ```bash
   cp .env.example .env
   ```
   *Note: Only `GEMINI_API_KEY` is strictly required for embeddings. Rerank keys can be configured in the UI.*

3. **Start the Stack:**
   ```bash
   docker-compose up -d
   ```

4. **Verify:**
   - **Dashboard:** http://localhost:3000
   - **API Health:** http://localhost:8081/health
   - **Weaviate:** http://localhost:8080/v1/meta

## Configuration

Configuration is managed via the **Settings** page in the UI or environment variables.

| Variable | Description | Default |
|----------|-------------|---------|
| `GEMINI_API_KEY` | Key for Google Gemini (Embeddings) | **Required** |
| `RERANK_PROVIDER` | `none`, `jina`, `cohere` | `none` |
| `RERANK_API_KEY` | API Key for selected provider | - |
| `SEARCH_ALPHA` | Hybrid search balance (0.0=Keyword, 1.0=Vector) | `0.5` |
| `SEARCH_TOP_K` | Max results to return | `5` |

## Architecture

Qurio follows a modular microservices architecture:

- **Frontend (Vue 3):** User interface for managing sources and settings.
- **Backend (Go):** API Gateway, MCP Server, and business logic.
- **Worker (Python):** Async ingestion engine handling crawling (`crawl4ai`) and parsing (`docling`).
- **Data Stores:**
  - **PostgreSQL:** Relational data (sources, jobs, settings).
  - **Weaviate:** Vector database for semantic search.
  - **NSQ:** Message queue for decoupling ingestion tasks.

## API Reference

### Sources

- `GET /api/sources`: List all sources.
- `POST /api/sources`: Add a new web source.
  ```json
  { "url": "https://example.com", "max_depth": 2 }
  ```
- `POST /api/sources/upload`: Upload a file source.
- `DELETE /api/sources/:id`: Delete a source and its data.
- `POST /api/sources/:id/resync`: Trigger a re-sync.

### Jobs

- `GET /api/jobs/failed`: List failed ingestion jobs.
- `POST /api/jobs/:id/retry`: Retry a failed job.

### Stats

- `GET /api/stats`: System overview counts.

### MCP (Model Context Protocol)

- `POST /mcp/messages`: Standard MCP JSON-RPC 2.0 endpoint.
- `GET /mcp/sse`: Server-Sent Events for MCP.

## License

MIT License. See [LICENSE](LICENSE) for details.
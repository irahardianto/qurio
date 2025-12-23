<div align="center">

<img src="docs/logo/qurio-inverted-black.png" alt="Qurio Logo" width="200"/>

# Qurio
**Know More, Hallucinate Less**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue.js-3.x-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![MCP](https://img.shields.io/badge/Protocol-MCP-orange)](https://modelcontextprotocol.io/)

<p align="center">
  <strong>The Open Source Knowledge Engine for AI Agents</strong><br>
  Built for localhost. Grounded in truth.
</p>

</div>

---

## 📖 About

**Qurio** is a self-hosted, open-source ingestion and retrieval engine that functions as a local **Shared Library** for AI agents (like Cursor, Windsurf, or custom scripts). 

Unlike cloud-based RAG solutions that introduce latency and privacy risks, Qurio runs locally to ingest your **handpicked** heterogeneous documentation (web crawls, PDFs, Markdown) and serves it directly to your IDE via the **Model Context Protocol (MCP)**. This ensures your AI writes better code faster using only the context you trust.

### Why Qurio?
*   **Privacy First:** Your data stays on your machine (`localhost` only).
*   **Precision:** Retrieves grounded "truth" to prevent AI hallucinations.
*   **Speed:** Deploys in minutes with `docker-compose`.
*   **Open Standards:** Built on MCP, Weaviate, and PostgreSQL.

## ✨ Key Features

- **🌐 Universal Ingestion:** Crawl documentation sites (with `llms.txt` support) or upload files (PDF, DOCX, MD).
- **🧠 Hybrid Search:** Combines BM25 keyword search with Vector embeddings (Gemini) for high-recall retrieval.
- **🎯 Configurable Reranking:** Integrate Jina AI or Cohere for precision tuning.
- **🔌 Native MCP Support:** Exposes a standard JSON-RPC 2.0 endpoint for seamless integration with AI coding assistants.
- **🕸️ Smart Crawling:** Recursive web crawling with depth control, regex exclusions, and sitemap support.
- **📄 OCR Pipeline:** Automatically extracts text from scanned PDFs and images via Docling.
- **🖥️ Admin Dashboard:** Manage sources, view ingestion status, and debug queries via a clean Vue.js interface.

## 🏗️ Architecture

Qurio is built as a set of microservices orchestrated by Docker Compose:

*   **Backend (Go):** Core orchestration, API, and MCP server.
*   **Frontend (Vue.js + Tailwind):** User interface for managing sources and settings.
*   **Ingestion Worker (Python):** Handles heavy lifting like crawling, OCR, and file processing.
*   **Vector Store (Weaviate):** Stores embeddings and handles hybrid search.
*   **Database (PostgreSQL):** Stores metadata, job status, and configuration.
*   **Queue (NSQ):** Manages asynchronous ingestion tasks.

## 🚀 Getting Started

### Prerequisites

*   [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
*   A [Google Gemini API Key](https://aistudio.google.com/app/apikey) (for embeddings)

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/yourusername/qurio.git
    cd qurio
    ```

2.  **Configure Environment:**
    Copy the example environment file and add your API key.
    ```bash
    cp .env.example .env
    ```

3.  **Start the System:**
    ```bash
    docker-compose up -d
    ```
    *Wait a minute for all services (Weaviate, Postgres) to initialize.*

4.  **Access the Dashboard:**
    Open [http://localhost:3000](http://localhost:3000) in your browser.

5. **Add API Keys:**
    Access [http://localhost:3000/settings](http://localhost:3000/settings) page in the dashboard, and add your Gemini and JinaAI/Cohoere(optional) API Keys


## 💡 Usage

### 1. Add Data Sources
Navigate to the Admin Dashboard ([http://localhost:3000](http://localhost:3000)) and click **"Add Source"**.
*   **Web Crawl:** Enter a documentation URL (e.g., `https://docs.encore.dev`). Configure depth and exclusion patterns.
*   **File Upload:** Drag and drop PDFs or Markdown files.

### 2. Connect Your AI Agent (MCP)
Configure your MCP-enabled editor (like Cursor) to connect to Qurio.

**For Cursor:**
Add the following to your MCP settings:
```json
{
  "mcpServers": {
    "qurio": {
      "type": "sse",
      "url": "http://localhost:8081/mcp/sse"
    }
  }
}
```
*Note: Direct HTTP transport for MCP is also supported at `http://localhost:8081/mcp` if your client supports it.*

### 3. Query
Ask your AI agent a question. It will now have access to the documentation you indexed!
> "How do I configure connection pooling in Encore?"

## 🗺️ Roadmap

- [x] Localhost Deployment
- [x] Web Crawling & `llms.txt` support
- [x] PDF/File Ingestion with OCR
- [x] MCP Endpoint
- [ ] Source Code Ingestion (AST analysis)
- [ ] GraphRAG Integration
- [ ] Multi-language Support

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details on how to submit pull requests, report issues, and request features.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  Built with ❤️ for the Developer Community
</p>

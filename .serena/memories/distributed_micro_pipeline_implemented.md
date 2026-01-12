# Distributed Micro-Pipeline Implementation

**Date:** 2026-01-12
**Status:** Implemented

## Overview
The ingestion pipeline has been refactored into a distributed micro-pipeline to decouple coordination from heavy lifting (embedding).

## Architecture Changes

### 1. Consumers
- **ResultConsumer (Coordinator)**:
  - Consumes `ingest.result` (from Docling/File upload).
  - Validates content.
  - Deletes old chunks (idempotency).
  - Splits content into chunks.
  - Publishes `IngestEmbedPayload` to `ingest.embed`.
  - Discovers links and publishes to `ingest.task.web`.
  - Does NOT embed or store chunks.
  
- **EmbedderConsumer (Worker)**:
  - Consumes `ingest.embed`.
  - Reconstructs contextual string (Title + Source + Content).
  - Calls `Embedder.Embed`.
  - Calls `VectorStore.StoreChunk`.
  - Independent scaling via `INGESTION_CONCURRENCY`.

### 2. Configuration Toggles
- `ENABLE_API`: Controls HTTP server startup.
- `ENABLE_EMBEDDER_WORKER`: Controls EmbedderConsumer startup.
- `INGESTION_CONCURRENCY`: Controls concurrency for EmbedderConsumer.

### 3. Deployment & Scaling
- **backend-api**: Runs with `ENABLE_API=true`, `ENABLE_EMBEDDER_WORKER=false`.
- **backend-worker**: Runs with `ENABLE_API=false`, `ENABLE_EMBEDDER_WORKER=true`.
- **backend** (Legacy/Full): Can run both if enabled.

#### Replica Configuration
Scaling is managed via Docker Compose replicas using environment variables:
- `INGESTION_WORKER_WEB_REPLICAS`: Web crawling workers (Default: 1).
- `INGESTION_WORKER_FILE_REPLICAS`: File processing workers (Default: 1).
- `BACKEND_WORKER_REPLICAS`: Embedding workers (Default: 1).

## Testing
- Integration tests (`apps/backend/internal/worker/integration_test.go`) now verify the full multi-hop pipeline (Result -> Embed -> Store).
- Unit tests updated to reflect decoupled responsibilities.

# Ingestion Worker Enhancements (2026-01-04)

## File Handler V2
- **Metadata Extraction:** Now extracts `title`, `author`, `created_at`, `pages`, `language` using Docling.
- **Concurrency:** Limited to 2 concurrent conversions using `asyncio.Semaphore`.
- **Timeout:** Enforced 300s timeout per file.
- **Error Handling:** Introduced `IngestionError` with structured codes:
  - `ERR_ENCRYPTED`: Password protected files.
  - `ERR_INVALID_FORMAT`: Unrecognized file formats.
  - `ERR_EMPTY`: No content extracted.
  - `ERR_TIMEOUT`: Processing exceeded 300s.

## Payload Structure
Success:
```json
{
  "source_id": "...",
  "status": "success",
  "content": "...",
  "metadata": {
    "title": "...",
    "author": "...",
    "created_at": "...",
    "pages": 10,
    ...
  },
  ...
}
```

## Backend Consumption (2026-01-04)
- **ResultConsumer:** Parses `metadata` from ingestion payload.
- **Contextual Embedding:** Enriched with `Author` and `Created` fields.
- **Storage:** Persists `author`, `createdAt`, `pageCount` to Weaviate `DocumentChunk`.
- **Parallel Processing:**
  - **Worker:** Uses `pebble.ProcessPool` (8 workers, 2 threads/worker) to isolate PDF conversion processes.
  - **Backend:** Uses a worker pool (concurrency: 5) to parallelize embedding and storage of chunks.
- **Protocol Limits:** NSQ and Backend Consumer configured for 50MB max message size to handle large PDFs.

Failure:
```json
{
  "source_id": "...",
  "status": "failed",
  "error": {
    "code": "ERR_CODE",
    "message": "Human readable message"
  },
  ...
}
```
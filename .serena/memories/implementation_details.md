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
    "pages": 10,
    ...
  },
  ...
}
```

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
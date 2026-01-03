# Enhancement Requirement: Enhanced Document Extraction & Metadata Strategy

**Date:** January 3, 2026  
**Status:** Draft  
**Target Component:** `apps/ingestion-worker` (File Handler)  

## 1. Executive Summary
The current document ingestion pipeline utilizes `Docling` for file-to-markdown conversion but treats all documents as flat text blobs. It fails to leverage the rich metadata available in modern file formats (PDF, DOCX) and lacks granular error handling for common user-facing issues (e.g., encryption). 

This enhancement aims to upgrade the file ingestion worker to extract semantic metadata (Title, Author, Creation Date) for better search ranking and context, and to implement "production-grade" error reporting to provide actionable feedback to users.

## 2. Objectives
1.  **Enrich Search Context:** Populate `source` and `chunk` metadata with authentic document properties (Title, Author) instead of relying solely on filenames.
2.  **Improve User Feedback:** Distinguish between system failures (retriable) and document validation errors (non-retriable, e.g., "Password Protected").
3.  **Ensure Stability:** Prevent resource exhaustion from large or malformed files via strict timeouts and memory safeguards.

## 3. Technical Requirements

### 3.1 Metadata Extraction
The worker must extract the following standard metadata fields from `Docling`'s internal model and map them to the result payload:

| Field | Source (Priority Order) | Fallback | Purpose |
| :--- | :--- | :--- | :--- |
| **Title** | `doc.meta.title` | Filename (cleaned) | Display in search results; boosting relevance. |
| **Author** | `doc.meta.author` / `doc.meta.creator` | `null` | Contextual filtering (e.g., "Files by John"). |
| **Created At** | `doc.meta.creation_date` | Upload timestamp | Temporal relevance sorting. |
| **Page Count** | `doc.num_pages` | `0` | Complexity estimation and user info. |
| **Language** | `doc.meta.language` | `en` (default) | Language-specific tokenizer optimization. |

### 3.2 Error Classification (Taxonomy)
The worker must catch specific exceptions and map them to standardized error codes in the `ingest.result` payload.

*   **`ERR_ENCRYPTED`**: File requires a password.
    *   *Action:* Stop. User must unlock file.
*   **`ERR_INVALID_FORMAT`**: File extension matches but header/content is corrupt.
    *   *Action:* Stop. User must re-generate file.
*   **`ERR_EMPTY`**: File contains no extractable text (e.g., image-only PDF without OCR enabled).
    *   *Action:* Warning/Stop. Suggest enabling OCR.
*   **`ERR_TIMEOUT`**: Processing exceeded 300s limit.
    *   *Action:* System Retry (1x). If fail again, mark as "Too Complex".

### 3.3 Resource Limits
*   **Timeout:** Hard limit of **300 seconds** per file.
*   **Concurrency:** Max **2 concurrent Docling processes** per worker instance (controlled via `ThreadPoolExecutor` or Semaphore) to prevent OOM on 2GB/4GB instances.

## 4. Architecture Changes

### `apps/ingestion-worker/handlers/file.py`
Refactor `handle_file_task` to:
1.  Initialize `DocumentConverter` with metadata extraction enabled.
2.  Wrap conversion in a `try/except` block matching the Error Taxonomy.
3.  Construct a `result` dictionary that includes a `metadata` object.

### Data Contract (`ingest.result` topic)
The JSON payload for successful ingestion will be expanded:

```json
{
  "source_id": "uuid",
  "status": "success",
  "content": "# Markdown Content...",
  "title": "Q3 Financial Report",
  "metadata": {
    "author": "Finance Dept",
    "pages": 12,
    "created_at": "2025-12-01T10:00:00Z",
    "filename": "Q3_Report_Final.pdf"
  }
}
```

## 5. Success Criteria
*   [ ] Uploading a PDF with a Title property displays that Title in the UI (or logs) instead of the filename.
*   [ ] Uploading a password-protected PDF results in a specific "Password Required" error message, not a generic "Worker Error".
*   [ ] Ingestion of valid 50-page PDFs completes successfully within 60 seconds.

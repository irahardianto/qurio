# Implementation Plan - MVP Part 3.6: Document Upload & OCR Integration

**Ref:** `2025-12-23-qurio-mvp-part3-6`
**Feature:** Document Ingestion (Upload, Storage, Worker Processing)
**Status:** Planned

## 1. Scope
Implement the end-to-end flow for uploading documents (PDF, Markdown, etc.), storing them in a shared volume, and processing them via the Ingestion Worker using Docling. This addresses the missing "File Upload" requirement from the MVP scope.

**Gap Analysis:**
- **Infrastructure:** `docker-compose.yml` uses risky `/tmp` host mapping. Needs named volume.
- **Backend:** Missing `POST /api/sources/upload` endpoint.
- **Frontend:** `SourceForm.vue` lacks file input.
- **Worker:** `handle_file_task` exists but needs to be verified against the shared volume path.

## 2. Requirements

### Functional
- **File Storage:** Uploaded files MUST be stored in a persistent Docker Volume shared between Backend and Worker.
- **Endpoint:** `POST /api/sources/upload` MUST accept `multipart/form-data`, save the file, create a `Source` record (Type: FILE), and publish an `ingest.task`.
- **Worker Access:** Worker MUST be able to read the file from the shared volume using the path provided in the NSQ task.
- **Deduplication:** Backend MUST calculate SHA-256 hash of the uploaded file and reject duplicates (as per PRD FR-2.3).

### Non-Functional
- **Reliability:** File storage must be atomic (save then process).
- **Security:** Validate file extensions/MIME types (allow .pdf, .md, .txt, .html). Max size 50MB.

## 3. Tasks

### Task 1: Infrastructure & Shared Volume
**Files:**
- Modify: `docker-compose.yml`

**Requirements:**
- **Acceptance Criteria**
  1. `docker-compose.yml` defines a named volume `qurio_uploads`.
  2. `backend` service mounts `qurio_uploads` to `/var/lib/qurio/uploads`.
  3. `ingestion-worker` service mounts `qurio_uploads` to `/var/lib/qurio/uploads`.

- **Functional Requirements**
  1. Replace host path `/tmp/qurio-uploads` with named volume.
  2. Ensure consistent mount point `/var/lib/qurio/uploads` in both services.

**Step 1: Implementation**
```yaml
# docker-compose.yml
volumes:
  qurio_uploads:
    name: qurio_uploads

services:
  ingestion-worker:
    volumes:
      - qurio_uploads:/var/lib/qurio/uploads

  backend:
    volumes:
      - qurio_uploads:/var/lib/qurio/uploads
```

**Step 2: Verify configuration**
Run: `docker-compose config`
Expected: Valid YAML output with volume definitions.

### Task 2: Backend Upload Handler
**Files:**
- Modify: `apps/backend/features/source/handler.go`
- Modify: `apps/backend/features/source/service.go`
- Modify: `apps/backend/features/source/source.go` (Add Type field if missing, or use inferred)
- Test: `apps/backend/features/source/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `Upload` handler accepts `multipart/form-data` with `file`.
  2. Saves file to `/var/lib/qurio/uploads/<uuid>_<filename>`.
  3. Calculates SHA-256.
  4. Calls `Service.Upload`.
  5. `Service.Upload` creates Source and publishes task with `path`.

- **Functional Requirements**
  1. `Handler.Upload`: MaxBytesReader (50MB).
  2. `Service.Upload`: Deduplication logic (check hash).
  3. `Service.Upload`: Publish message `{"type": "file", "path": "...", "id": "..."}`.

**Step 1: Write failing test**
```go
// apps/backend/features/source/handler_test.go
func TestUpload_Success(t *testing.T) {
    // Setup Mock Service
    // Create multipart request with "test.pdf"
    // Call handler
    // Assert status 201
    // Assert Service.Upload called
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/source/...`

**Step 3: Implementation**
```go
// handler.go
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
    r.ParseMultipartForm(50 << 20) // 50MB
    file, header, err := r.FormFile("file")
    // ... Save to disk ...
    // ... Calculate Hash ...
    // ... Call Service.Upload ...
}

// service.go
func (s *Service) Upload(ctx context.Context, filename string, path string, hash string) (*Source, error) {
    // Check duplicate
    // Create Source (Type: File)
    // Publish Task
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/source/...`

### Task 3: Frontend File Upload UI
**Files:**
- Modify: `apps/frontend/src/features/sources/SourceForm.vue`
- Modify: `apps/frontend/src/features/sources/source.store.ts`

**Requirements:**
- **Acceptance Criteria**
  1. UI provides a tab/switch for "Web" vs "File".
  2. File input accepts `.pdf, .md, .txt`.
  3. "Ingest" button uploads file via `POST /api/sources/upload`.
  4. Shows upload progress/error.

- **Functional Requirements**
  1. `source.store.ts`: Add `uploadSource(file: File)`.
  2. `SourceForm.vue`: Add Tabs (shadcn/ui or simple div toggle).

**Step 1: Implementation**
```typescript
// source.store.ts
async uploadSource(file: File) {
    const formData = new FormData()
    formData.append('file', file)
    // POST /api/sources/upload
}
```

**Step 2: Verify implementation**
Run: Manual verification (or component test if setup)

### Task 4: Worker File Processing Verification
**Files:**
- Modify: `apps/ingestion-worker/main.py`
- Test: `apps/ingestion-worker/tests/test_handlers.py`

**Requirements:**
- **Acceptance Criteria**
  1. Worker receives `type: file` task.
  2. Worker reads from correct path `/var/lib/qurio/uploads/...`.
  3. `handle_file_task` processes it.

- **Functional Requirements**
  1. Ensure `process_message` handles `type="file"` correctly (already present, verify path logic).

**Step 1: Verify Logic**
Review `apps/ingestion-worker/main.py`. Ensure it passes `path` from payload to `handle_file_task`.

**Step 2: Integration Test (Manual)**
1. `docker-compose up`
2. Upload file via UI.
3. Check backend logs (Saved file).
4. Check worker logs (Processing file at `/var/lib/qurio/uploads/...`).

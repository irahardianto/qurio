# Implementation Plan - MVP Part 3.4: Configuration Consistency & Frontend Integration

**Ref:** `2025-12-23-qurio-mvp-part3-4`
**Feature:** Configuration & Frontend
**Status:** Completed

## 1. Scope
Address the architectural inconsistency where the Python Worker relies on Environment Variables while the Backend uses DB-stored Settings. Ensure the Worker receives the dynamic API Key from the DB via the NSQ payload. Then, complete the Frontend integration for Source Management.

**Gap Analysis:**
- **Architecture:** Worker ignores DB Settings (`GEMINI_API_KEY`).
- **Backend:** `SourceService` does not fetch/pass keys to Worker.
- **Frontend:** `SourceForm` is not connected to real API.
- **Frontend:** `SourceList` might need polling/updates.

## 2. Requirements

### Functional
- **Config Propagation:** Worker uses the `GeminiAPIKey` defined in the Settings page (DB), passed via NSQ task payload.
- **Standardization:** The System relies on the DB-stored API Key. The Worker purely executes based on the task payload.
- **Source Management:** Users can Create, List, and Delete sources via UI.
- **Validation:** UI handles backend validation errors (e.g., Duplicates).

### Non-Functional
- **Security:** API Key is passed in internal NSQ payload (secured network), not exposed to client.

## 3. Tasks

### Task 1: Backend Settings Injection
**Files:**
- Modify: `apps/backend/features/source/source.go`
- Modify: `apps/backend/main.go`
- Test: `apps/backend/features/source/source_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `SourceService` receives `SettingsService` via Dependency Injection.
  2. `Create` and `ReSync` methods fetch `gemini_api_key` from `SettingsService`.
  3. NSQ payload includes `gemini_api_key`.
  4. Endpoints remain at `/sources` (mapped from `/api/sources` by Nginx).

- **Functional Requirements**
  1. Inject `settings.Service` into `source.Service` factory.
  2. In `source.go` (Create/ReSync):
     - Fetch settings: `s.settings.Get(ctx)`
     - Add `gemini_api_key` to `ingest.task` payload.

- **Test Coverage**
  - [Unit] `TestCreate_WithSettings` - verify settings service is called and key is in payload.

**Step 1: Write failing test**
```go
// apps/backend/features/source/source_test.go
func TestCreate_WithSettings(t *testing.T) {
    // Setup mock SettingsService returning "test-key"
    // Setup mock Publisher
    // Call Create
    // Assert Publisher.Publish argument contains "gemini_api_key": "test-key"
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/source/...`

**Step 3: Implementation**
```go
// source.go
type Service struct {
    // ...
    settings *settings.Service
}
// Update NewService signature
func NewService(repo Repository, pub EventPublisher, chunkStore ChunkStore, settings *settings.Service) *Service {
    return &Service{..., settings: settings}
}

func (s *Service) Create(ctx context.Context, src *Source) error {
    // ...
    set, err := s.settings.Get(ctx)
    apiKey := ""
    if err == nil && set != nil {
        apiKey = set.GeminiAPIKey
    }
    
    payload := map[string]interface{}{
        // ...
        "gemini_api_key": apiKey,
    }
    // ...
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/source/...`

### Task 2: Worker Key Usage
**Files:**
- Modify: `apps/ingestion-worker/main.py`
- Modify: `apps/ingestion-worker/handlers/web.py`

**Requirements:**
- **Acceptance Criteria**
  1. Worker extracts `gemini_api_key` from NSQ message.
  2. `LLMContentFilter` uses this key.

- **Functional Requirements**
  1. `process_message` extracts `gemini_api_key`.
  2. `handle_web_task` accepts `api_key`.
  3. `LLMContentFilter` initialized with `api_token=api_key`.

**Step 1: Implementation**
```python
# handlers/web.py
async def handle_web_task(url: str, max_depth: int = 0, exclusions: list = None, api_key: str = None):
    # ...
    llm_filter = LLMContentFilter(..., api_token=api_key)
```

**Step 2: Verify implementation**
Run: `python3 -m pytest apps/ingestion-worker/tests/test_handlers.py` (ensure basic structure works, manual E2E required for integration)

### Task 3: Frontend Source Store Integration
**Files:**
- Modify: `apps/frontend/src/features/sources/source.store.ts`
- Modify: `apps/frontend/src/features/sources/SourceForm.vue`

**Requirements:**
- **Acceptance Criteria**
  1. `fetchSources` calls `GET /api/sources`.
  2. `addSource` calls `POST /api/sources` with `max_depth` (number) and `exclusions` (string[]).
  3. `deleteSource` calls `DELETE /api/sources/:id`.
  4. `resyncSource` calls `POST /api/sources/:id/resync`.

- **Functional Requirements**
  1. Ensure `source.store.ts` uses correct paths (no `/v1`).
  2. Update `addSource` payload construction to include new fields.

**Step 1: Implementation**
```typescript
// source.store.ts
// Verify paths are /api/sources...
// Update addSource signature/payload
```

**Step 2: Verify implementation**
Run: `npm run test:unit src/features/sources/source.store.spec.ts` (or similar unit test command)

### Task 4: End-to-End Verification
**Requirements:**
- **Acceptance Criteria**
  1. Set API Key in Settings (DB).
  2. Create Source via UI.
  3. Verify Worker log shows receipt of key and successful processing.

**Step 1: Execution**
1. Set Settings Key: `PUT /api/settings` {"gemini_api_key": "valid-key"}
2. Create Source via UI (or `curl`).
3. Check Worker logs for key usage.
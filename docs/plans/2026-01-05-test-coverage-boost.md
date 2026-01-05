# Test Coverage Boost Plan - Jan 5, 2026

## Objective
Increase code coverage to >90% across Backend, Frontend, and Worker.

## Strategy
Focus on high-value business logic and uncovered handlers/adapters.

## Tasks

### Backend (Go)
- [ ] **Settings Handler**: Create `apps/backend/internal/settings/handler_test.go`.
    - Test `GetSettings` (Success, Error).
    - Test `UpdateSettings` (Success, Validation Error).
    - Mock `SettingsService`.
- [ ] **Vector Adapter**: Create `apps/backend/internal/vector/adapter_test.go`.
    - Mock Weaviate client.
    - Test `AddObjects`, `Search`.
- [ ] **Gemini Dynamic Embedder**: Create `apps/backend/internal/adapter/gemini/dynamic_embedder_test.go`.
    - Test logic for key switching.

### Frontend (Vue/Vitest)
- [ ] **Settings Store**: Create `apps/frontend/src/features/settings/settings.store.spec.ts`.
    - Test actions (fetch, update).
    - Mock API client.
- [ ] **Source Progress Component**: Create `apps/frontend/src/features/sources/SourceProgress.spec.ts`.
    - Test progress bar rendering.
    - Test state (processing, completed, failed).

### Ingestion Worker (Python/Pytest)
- [ ] **Handlers**: Audit and add tests for edge cases in `apps/ingestion-worker/handlers`. (Deferred to Phase 2 if needed).

## Execution
Executing using `editing` mode tools.

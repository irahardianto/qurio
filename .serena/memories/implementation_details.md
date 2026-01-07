### 2026-01-07: Backend Test Coverage Improvements
- **Bootstrap Refactoring:**
    - Extracted retry logic for Weaviate schema initialization into `EnsureSchemaWithRetry` (pure function with retry loop).
    - Updated `Bootstrap` to use `EnsureSchemaWithRetry`.
    - Added unit tests for retry logic using `MockVectorStore`.
- **Integration Testing:**
    - Implemented `apps/backend/internal/app/bootstrap_integration_test.go` using `Testcontainers`.
    - Updated `IntegrationSuite` to expose container configuration via `GetAppConfig`.
    - Added support for running migrations in `IntegrationSuite` or deferring to `Bootstrap` (via `SkipMigrations` flag).
    - Added `MigrationPath` to `config.Config` to allow tests to override migration location.
- **Application Structure:**
    - Refactored `apps/backend/main.go` to extract `run(ctx, cfg, logger)` function.
    - Added `apps/backend/smoke_test.go` (package `main`) to verify full application startup and wiring by running `run` against Testcontainers.
    - Enhanced `app.New` unit tests to verify route registration for key endpoints.
- **Feature Hardening:**
    - **Source:** Implemented `Exclusions` regex validation in `Service.Create`. Added `POST /sources/upload` integration test verifying file persistence to `QURIO_UPLOAD_DIR`.
    - **Job:** Verified `Retry` timeout logic (5s limit on NSQ publish). Validated `failed_jobs` cascade delete via integration tests.
    - **MCP:** Hardened JSON-RPC handler with table-driven tests for edge cases (empty params, invalid values). Validated SSE session establishment and Correlation ID propagation in integration tests.

$1

$1
- **Ingestion Worker Fix:**
  - Diagnosed `Permission denied` error in `RapidOCR` model download.
  - Updated `apps/ingestion-worker/Dockerfile` to run model download as `root` (before switching user) to ensure write access to `site-packages`.
$1
$1
$1
- **Backend Configuration:**
  - Added `GeminiAPIKey` to `config.Config` (read from `GEMINI_API_KEY`).
  - Updated `app.New` to seed the database settings with the environment-provided API key if the setting is empty.
  - This resolves the "gemini api key not configured" error during embedding if the UI setup hasn't been completed yet.

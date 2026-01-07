- **Database Constraints & Testing:** Partial unique indexes (e.g., `WHERE deleted_at IS NULL`) and constraints like `ON DELETE CASCADE` cannot be reliably tested with `sqlmock`. They require real database integration tests (using Testcontainers) to verify behavior correctness.
- **MCP Testing:** Testing async SSE flows requires integration tests to verify context propagation (Correlation IDs) as unit tests often miss the goroutine context detachment/cancellation nuances.
- **Handler Integration Tests:** For handlers involving multipart uploads or complex state (like SSE sessions), use `httptest` combined with `IntegrationSuite`. This allows verifying side effects (files on disk, DB state) which unit tests with mocks cannot cover.
- **Testcontainers Pattern:**
    - Use `IntegrationSuite` struct to manage Postgres, Weaviate, and NSQ containers.
    - Setup in `TestMain` or per-test `Setup()` to ensure isolation.
    - Use `testcontainers.WithWaitStrategy` to ensure services are fully ready before running tests.
    - Expose dynamic ports via `GetAppConfig()` to configure clients/handlers under test.
- **Ingestion Worker Testing:**
    - **PYTHONPATH:** When running `pytest` for the ingestion worker, explicitly set `PYTHONPATH=.` (e.g., `PYTHONPATH=. ./venv/bin/pytest`) to ensure local modules (like `handlers`, `config`, `logger`) are correctly resolved.
    - **Dead Code:** Unused imports and variables (e.g., redundant semaphores, unused `exclusions` params) can accumulate. Periodic cleanup using static analysis or manual review is recommended to keep the worker codebase lean.

### Docker & Ingestion Reliability (2026-01-07)
- **Shared Volume Permissions:**
    - **Issue:** When sharing a volume between containers (e.g., `backend` and `ingestion-worker`), UID mismatches cause `Permission denied` errors. Alpine uses variable UIDs for `adduser`, while Debian uses `1000`.
    - **Fix:** Explicitly set UIDs (e.g., `RUN adduser -u 1000 ...`) in Dockerfiles.
- **Model Caching & Non-Root Users:**
    - **Issue:** Python libraries like `RapidOCR` and `Docling` may try to download models to `site-packages` or user home directories at runtime. This fails in non-root containers if the directory isn't writable or if the library ignores cache env vars.
    - **Fix:**
        1. Set explicit cache paths: `ENV HF_HOME=/app/.cache`.
        2. Create and `chown` these directories to the non-root user.
        3. Run download scripts as `root` *during build* to populate these shared caches.
        4. If a library insists on using `site-packages` (like `rapidocr-onnxruntime`), explicitly `chown` that specific package directory to the non-root user.
- **Configuration Seeding:**
    - **Issue:** Ingestion fails silently (or with logs only) if the embedding API Key is missing from the database settings, even if the environment variable is set.
    - **Fix:** Implement logic in the application bootstrap (`app.New`) to seed database settings from environment variables (`GEMINI_API_KEY`) if the setting is currently empty. This ensures a "zero-config" start for fresh deployments.

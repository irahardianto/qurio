# Core Foundation State

## Codebase Health & Coverage (Jan 2026)

### Test Coverage Stats
- **Backend (Go):** **100% Pass Rate**
    - **Core Logic:** High coverage in `internal/text`, `internal/settings`, `internal/vector`.
    - **Wiring:** Refactored `main.go` into `internal/app` with **Dependency Injection**, enabling full wiring tests (`TestNew_Success`, `TestNew_PanicsOnInvalidDB`).
    - **Bootstrap:** `Bootstrap` function covers infrastructure initialization and retry logic (`TestBootstrap_ConfigurationError`).
    - **Adapters:** Critical paths in `Gemini` (Key Rotation), `Weaviate` (Store/Search, Network Errors), and `NSQ` are fully tested.
    - **Handlers:** Standardized error handling (404/Not Found) verified in `source` and `job` features. MCP Handler fully covered via table-driven tests.
    - **Workers:** `ResultConsumer` hardened with tests for invalid JSON and dependency failures.
- **Frontend (Vue):** **100% Pass Rate** (64/64 Tests)
    - **UI Library:** Comprehensive tests for Shadcn wrappers (`Select`, `Card`, `Badge`) including complex portal interactions.
    - **Stores:** Robust error handling and edge case coverage for `source`, `settings`, `job`, and `stats`.
    - **Components:** Verified loading states, error messages, and form resets in `Settings.vue` and `SourceForm.vue`.
- **Ingestion Worker (Python):** **100% Pass Rate** (17/17 Tests)
    - Verified web crawling, file handling, and messaging reliability.

### Verified Implementations
1. **API Envelopes:** Standardized JSON envelope format implemented across handlers.
2. **Background Janitor:** Implemented and operational in `main.go`.
3. **MCP Context:** Correlation IDs properly propagated in contexts (fixed trace chain abandonment).
4. **Data Consistency:** `updated_at` vs `lastSyncedAt` resolved across stack.
5. **Configuration:** No redundancy in TSConfig.
6. **Architecture Refactor:** `internal/app` now uses **Interfaces** (`Database`, `VectorStore`, `TaskPublisher`) for I/O isolation, resolving previous coverage gaps.
7. **Bootstrap Decoupling:** Infrastructure setup isolated in `bootstrap.go`, simplifying `main.go` and enabling targeted testing of initialization logic.

## Critical Subsystems
- **Ingestion:** Robust pipeline with `ChunkMarkdown` (93% covered) and `ResultConsumer` (High coverage).
- **Search:** Hybrid search implementation verified in `internal/retrieval`.
- **Vector Store:** Weaviate adapter fully mocked and tested (`internal/adapter/weaviate`).
- **Resilience:** Backend handles missing keys, DB retries, and schema initialization automatically.

## Known Issues (Jan 6, 2026)
- **None critical.** Previous issues with backend coverage and handler errors were resolved in `2026-01-06-test-coverage-boost-1.md` and `2026-01-06-test-coverage-boost-2.md` executions.

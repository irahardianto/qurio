# Core Foundation State

## Codebase Health & Coverage (Jan 2026)

### Status
The codebase is in a robust state with significantly improved test coverage following the Jan 5th, 2026 sprint. The system implementation is ahead of older documentation.

### Test Coverage Stats
- **Frontend (Vue):** **87.5%** (Target: >90%)
    - Critical stores (`source`, `settings`, `stats`, `jobs`) are >90% covered.
    - Key components (`SourceForm`, `SourceList`, `SourceProgress`) are >88% covered.
    - Remaining gap: `components/ui` (shadcn wrappers) at ~78%.
- **Backend (Go):** **55.7%** (Statements)
    - **Core Logic:** High coverage in `internal/text` (93%), `internal/settings` (93%), `internal/vector` (79%).
    - **Features:** `features/source` (64%), `features/job` (59%).
    - **Skew:** Low overall percentage due to `main.go` (0%) containing wiring logic.

### Verified Implementations
1. **API Envelopes:** Standardized JSON envelope format implemented across handlers.
2. **Background Janitor:** Implemented and operational.
3. **MCP Context:** Correlation IDs properly propagated in contexts.
4. **Data Consistency:** `updated_at` vs `lastSyncedAt` resolved across stack.
5. **Configuration:** No redundancy in TSConfig.

## Critical Subsystems
- **Ingestion:** Robust pipeline with `ChunkMarkdown` (93% covered) and `ResultConsumer` (70% covered).
- **Search:** Hybrid search implementation verified in `internal/retrieval`.
- **Vector Store:** Weaviate adapter fully mocked and tested (`internal/adapter/weaviate`).

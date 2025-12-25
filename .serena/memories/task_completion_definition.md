# Task Completion Definition

## Done Criteria
A task is considered done when:
1.  **Code Implemented:** The planned code changes are applied.
2.  **Tests Passed:**
    -   Unit tests for new/modified logic passed.
    -   Integration tests for DB/API interactions passed.
    -   Relevant E2E tests (if any) passed.
3.  **Documentation Updated:**
    -   `README.md` updated if usage changed.
    -   API docs updated if endpoints changed.
    -   Architecture diagrams updated if system structure changed.
4.  **Verification:** The `verify_infra.sh` script passes (if relevant).
5.  **Clean Code:** No linting errors, no commented-out legacy code.

## Current Phase: MVP Polish & Verification
We are in the final stages of the MVP.
- **Part 1-3:** Core Features (Ingestion, Search, Worker, Reranking) - COMPLETE.
- **Part 4:** Configuration, Verification, E2E Testing - IN PROGRESS.

## Pending Tasks (from Part 4.1 plan)
-   [ ] Migration: Add `search_alpha` and `search_top_k`.
-   [ ] Backend: Update Settings and Retrieval services.
-   [ ] Frontend: Update Settings UI.
-   [ ] E2E: Add Search and Settings tests.

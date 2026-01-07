1. qurio/apps/backend/features/stats
• Assessment: This directory contains the simplest logic in the backend, consisting of a handler that aggregates counts from three repositories.
• Improvement Strategy: Unit Tests (High Priority).
• Actionable Detail: You do not need Testcontainers for this. Coverage is likely low because the current TestHandler_GetStats only covers a 200 OK scenario,. To bridge the gap, implement table-driven tests that mock individual repository failures (e.g., JobRepo.Count returning a db_error) to verify that the handler correctly triggers the writeError helper and returns the standard JSON error envelope,.
2. qurio/apps/backend/features/job
• Assessment: This feature manages the Dead Letter Queue (DLQ) for failed ingestion tasks and handles the critical "Retry" operation,.
• Improvement Strategy: Hybrid Strategy.
• Unit Component: The service.go file includes a complex select block with a 5-second timeout for NSQ publishing. Improve coverage by using a MockPublisher with a variable sleep duration to explicitly trigger the "timeout waiting for NSQ publish" error path.
• Integration Component: The failed_jobs table uses a REFERENCES sources(id) ON DELETE CASCADE constraint and UUID generation. Use Testcontainers for PostgreSQL to verify that jobs are correctly purged when a source is deleted and that the ORDER BY created_at DESC logic in the List method works with real database timestamps,.
3. qurio/apps/backend/features/source
• Assessment: This is the most complex directory, handling multi-part file uploads, recursive crawl configurations, and transactional bulk page creation,.
• Improvement Strategy: Integration-Heavy Mix.
• Integration Component (Uploads): The Upload handler involves heavy interaction with the file system (/var/lib/qurio/uploads) and SHA-256 calculation,. Mocking os.Create or io.Copy is brittle; instead, use integration tests to verify the full flow from multipart/form-data input to file persistence.
• Integration Component (Database): The repository uses a partial unique index (sources_content_hash_active_idx) that allows duplicate hashes only if the previous record was soft-deleted. You must use a real database to test this constraint logic, as it cannot be replicated by sqlmock.
• Unit Component: Improve the source_test.go file by adding edge cases for the Exclusions regex validation logic in Service.Create.
4. qurio/apps/backend/features/mcp
• Assessment: This directory implements the Model Context Protocol, involving massive JSON-RPC switch blocks, SSE transport, and context detachment,.
• Improvement Strategy: Unit-Heavy Strategy.
• Unit Component (Switch Blocks): Coverage is suppressed because the existing table-driven tests do not cover all tools (e.g., qurio_list_sources, qurio_read_page) and their negative paths,. You must expand the TestHandler_ProcessRequest_Table to include cases for:
    ◦ Invalid alpha values (outside 0.0–1.0).
    ◦ Malformed filters objects.
    ◦ Tool-specific failures like qurio_read_page receiving an empty URL.
• Integration Component (Traceability): The SSE implementation (HandleSSE) and async goroutines in HandleMessage use context.WithoutCancel,. Use an integration test to verify that the Correlation ID is correctly preserved and logged throughout the async tool execution lifecycle,.

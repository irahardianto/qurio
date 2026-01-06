This technical scope outlines the complete implementation of integration tests for the apps/backend application using Testcontainers. This plan ensures 100% adherence to the Technical Constitution Rule 1 (I/O Isolation) and explicitly addresses the "testability gaps" identified in the backend forensic reports.
1. Infrastructure Requirements (Containers)
The integration suite must provision three ephemeral containers to mirror the production docker-compose.yml environment:
• PostgreSQL Container:
    ◦ Image: postgres:16-alpine.
    ◦ Port Mapping: Dynamic (mapped from 5432).
    ◦ Data Initialization: Must execute all SQL migrations found in apps/backend/migrations/*.sql using the project's migration driver.
    ◦ State: Cleaned or dropped between test suites to ensure idempotency.
• Weaviate Container:
    ◦ Image: semitechnologies/weaviate:latest.
    ◦ Environment: AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: 'true', DEFAULT_VECTORIZER_MODULE: 'none'.
    ◦ Port Mapping: Dynamic (mapped from 8080).
    ◦ Initialization: Must call EnsureSchema via the internal/vector package to establish the DocumentChunk class and properties.
• NSQ Container:
    ◦ Image: nsqio/nsq:v1.3.0.
    ◦ Components: Combined nsqd and nsqlookupd configuration.
    ◦ Port Mapping: 4150 (TCP) and 4151 (HTTP).
    ◦ Pre-flight: Must call createTopics logic from bootstrap.go to ensure ingest.task and ingest.result exist.

--------------------------------------------------------------------------------
2. Implementation Phase 1: Shared Test Suite Scaffolding
To avoid code duplication and manage container lifecycles, a central test helper must be implemented.
• Files to Create: apps/backend/internal/testutils/integration_suite.go.
• Core Logic:
    ◦ Define a Suite struct containing pointers to *sql.DB, *weaviate.Client, and *nsq.Producer.
    ◦ Implement a SetupSuite() function using testcontainers-go to pull images and wait for "Ready" strategies (e.g., wait.ForLog or wait.ForSQL).
    ◦ Implement a TeardownSuite() to purge all containers.
• Context Management: All container operations must use a 60-second hard timeout to prevent CI hangs.

--------------------------------------------------------------------------------
3. Implementation Phase 2: Feature Repository Integration
These tests verify that Go structs correctly map to SQL/Vector properties without tokenization or constraint errors.
A. Source Feature Integration
• Target: apps/backend/features/source/repo_integration_test.go.
• Scenarios:
    ◦ Deduplication: Insert a Source, then attempt to insert another with the same content_hash; verify PostgreSQL rejects it.
    ◦ Soft Delete: Call SoftDelete and verify the record remains in the DB but deleted_at is populated.
    ◦ Page Management: Bulk insert 100 SourcePage records and verify CountPendingPages returns 100.
B. Vector Store Integration
• Target: apps/backend/internal/adapter/weaviate/store_integration_test.go.
• Scenarios:
    ◦ Exact Match Deletion: Store a chunk with a specific URL, call DeleteChunksByURL, and verify it is gone (validates the fix from text to string data types).
    ◦ Hybrid Search: Store two chunks—one with the word "Postgres" and one about "Databases". Query for "Postgres" with alpha=0.0 (Keyword) and verify the first chunk is ranked higher.
    ◦ Metadata Retrieval: Verify that Search returns top-level fields for Author, CreatedAt, and PageCount.

--------------------------------------------------------------------------------
4. Implementation Phase 3: ResultConsumer Orchestration
This is the "critical logic hub" that currently lacks integration coverage.
• Target: apps/backend/internal/worker/integration_test.go.
• Implementation Steps:
    1. Remove t.Skip: Un-skip the TestIngestIntegration function.
    2. Full Flow Simulation:
        ▪ Mock the Gemini Embedder (to avoid external API costs/keys) but use a real Weaviate container.
        ▪ Push a mock ingest.result message into the real NSQ container.
        ▪ Trigger ResultConsumer.HandleMessage.
        ▪ Validation: Check the real PostgreSQL source_pages table to ensure the status moved to completed.
        ▪ Validation: Check the real Weaviate instance to ensure the chunk was actually persisted with the correct sourceName.

--------------------------------------------------------------------------------
5. Implementation Phase 4: MCP Tool Integration
Ensures the AI agent receives grounded, valid JSON responses.
• Target: apps/backend/features/mcp/handler_integration_test.go.
• Scenarios:
    ◦ qurio_read_page: Seed the DB with 5 chunks for one URL. Call the tool and verify the output is a single, concatenated Markdown block sorted by chunkIndex.
    ◦ qurio_search with Filters: Call search with filters: { "type": "code" } and verify only code-type chunks are returned from the container.
    ◦ Correlation ID: Send an MCP request with an X-Correlation-ID header and verify the same ID appears in the resulting container logs or DB metadata.

--------------------------------------------------------------------------------
6. GitHub Actions (CI) Configuration
To ensure these tests run in the automated pipeline, the .github/workflows/test.yml must be updated.
• Job Requirements:
    ◦ Standard ubuntu-latest runner (includes Docker).
    ◦ Environment: Set TEST_CONTAINERS_ENABLED=true.
• Resource Tuning:
    ◦ Because Weaviate and Go compilation are heavy, limit the NSQ concurrency during integration tests to avoid OOM.
    ◦ Add a step to verify_infra.sh that checks for Docker daemon availability before starting the Go test suite.

--------------------------------------------------------------------------------
Analogy for Integration Testing Unit testing was like making sure every lego brick was the right shape in the factory. Integration testing with Testcontainers is like actually snapping the bricks together to build the model. Right now, your factory is covered Unit tests, but the "instruction manual" (ResultConsumer/Bootstrap) hasn't been tested to see if the bricks actually fit together under the weight of the real world. Testcontainers provides the baseplate for that build.
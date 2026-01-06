Based on a forensic analysis of the apps/backend/ folder and recent coverage reports, the backend statement coverage remains at 55.7%. Although core logic packages like internal/text and internal/settings have reached >90%, the overall percentage is suppressed by 0% coverage in main.go and significant "testability gaps" in the orchestration and feature layers.
The following comprehensive plan is required to reach the target of 95% unit test coverage by enforcing Technical Constitution Rule 1 (I/O Isolation) across all remaining components.
1. Decoupling and Testing the "Glue" Logic
The primary reason for low coverage is that the application's wiring logic still relies on live infrastructure initialization during unit tests.
• Refactor apps/backend/internal/app/app.go: While the App struct now uses interfaces (Database, VectorStore, TaskPublisher), the constructor app.New is currently tested by passing live or partially mocked clients that still attempt network connections.
• Action Plan:
    1. Fully implement the Mocking Strategy in apps/backend/internal/app/mocks_test.go.
    2. Update app_test.go to use these mocks to verify that every route in the http.NewServeMux() is correctly registered and that the middleware stack is applied without initiating a database ping or Weaviate schema check.
    3. Move the remaining retry loops for DB and Weaviate from the initialization flow into a separate Bootstrap package that returns the Dependencies interface, allowing main.go to remain a "thin" entry point.
2. Exhaustive Table-Driven Testing for MCP Tools
The apps/backend/features/mcp/handler.go file is a large source of uncovered logic due to its complex switch block and JSON-RPC unmarshaling.
• Current Gap: Individual tools like qurio_search, qurio_read_page, and qurio_list_sources have basic "happy path" tests, but negative paths for invalid arguments or internal service failures are missing.
• Action Plan:
    1. Implement a comprehensive table-driven test suite for processRequest in apps/backend/features/mcp/handler_test.go.
    2. Test Cases to Add:
        ▪ Initialize: Verify protocol version and capabilities.
        ▪ List Tools: Ensure all qurio_ prefixed tools are returned with correct descriptions.
        ▪ Call Unknown Tool: Explicitly test the ErrMethodNotFound (-32601) code.
        ▪ Search Arguments: Test missing query (required), invalid alpha range (outside 0.0-1.0), and malformed filters object.
        ▪ Read Page: Test missing url and cases where GetChunksByURL returns zero results.
3. Hardening the Ingestion ResultConsumer
The ResultConsumer (currently 70% covered) contains "micro-logic" decisions in its HandleMessage loop that are not fully exercised.
• Current Gap: Tests do not simulate "Poison Pill" messages (corrupted JSON) or partial failures where the Embedder fails but the PageManager should still update status.
• Action Plan:
    1. Add unit tests in apps/backend/internal/worker/result_consumer_test.go for corrupted NSQ bodies.
    2. Error Injection: Configure MockEmbedder and MockVectorStore to return specific errors (api error, connection refused) to verify that the consumer correctly triggers a requeue or saves a record to failed_jobs without crashing the process.
    3. Timeout Verification: Explicitly test that the 60-second context timeout for embedding and storage is correctly applied.
4. Closing Gaps in Feature Handlers (HTTP Response Logic)
Handlers for source and job features have coverage gaps in their HTTP response and validation paths.
• Current Gap: Paths for sql.ErrNoRows leading to 404 Not Found responses are not consistently tested.
• Action Plan:
    1. Write tests for SourceHandler.Get and JobHandler.Retry that explicitly trigger the writeError call for non-existent IDs.
    2. MIME/Size Validation: In apps/backend/features/source/handler.go, write unit tests that simulate a multipart/form-data request with a file exceeding 50MB and unsupported extensions to cover early return paths.
5. Adapter Network Error Simulation
Adapters for Weaviate and Gemini lack coverage for retry logic and network-level edge cases.
• Current Gap: If the Weaviate API returns a 503 Service Unavailable or a GraphQL error, the current tests do not verify the adapter's error handling.
• Action Plan:
    1. Use httptest.NewServer in apps/backend/internal/adapter/weaviate/store_test.go to return specific HTTP status codes (500, 503) and GraphQL error JSON.
    2. Implement Key Rotation tests for the Gemini.DynamicEmbedder to ensure it correctly switches keys when the Settings repository updates.

--------------------------------------------------------------------------------
Analogy: Reaching 95% coverage is like waterproofing a ship. Currently, your ship has a strong hull in the center (Core Logic), but the engine room (main.go) is completely open to the elements, and there are dozens of tiny, unsealed rivets (Error Paths) throughout the deck. To sail safely in high-pressure production environments, you must seal every tiny gap where an error could leak in, ensuring that even if one room floods (a Service Failure), your bulkheads (Error Handling) are tested and ready to hold.
Based on an analysis of the current code within the apps/ directory, the following inconsistencies and deviations from the Technical Constitution remain present. While many issues (such as Weaviate schema tokenization and raw API envelopes) have been addressed in recent refactors, several structural and functional misalignments persist.
1. MCP SSE Trace Chain Breakage
The project mandates that a correlationId must be generated at ingress and propagated throughout the system for traceability.
• The Inconsistency: In apps/backend/features/mcp/handler.go, the HandleMessage method (used for SSE transport) correctly extracts the correlationID from the request context. However, the actual tool execution is triggered within an asynchronous goroutine that explicitly discards this context.
• Evidence: The code contains a TODO-style comment: "Create a new context with correlation ID if we had a way to propagate it easily / For now just pass background context". By using context.Background(), any logs or sub-operations (like retrieval) triggered by this request will lose their association with the original trace.
2. Ingestion Worker Handler Contract
There is a functional inconsistency in the communication contract between the individual handlers and the main dispatcher loop in the Python worker.
• The Inconsistency: handle_web_task returns a single dictionary representing the crawl result. In contrast, handle_file_task returns a list of dictionaries.
• The Result: This forces the main.py dispatcher to use inconsistent wrapping logic: it must manually wrap the web result in a list (results_list = [result]) while treating the file result as a ready-to-iterate list. This creates a fragile internal API that complicates the addition of future handlers.
3. Missing Background Reliability Orchestration
The "Ingestion Robustness Diagnosis" identified a critical need for a "Janitor" mechanism to rescue jobs that crash silently and remain in a processing state forever.
• The Inconsistency: While the low-level logic has been implemented in the repository (ResetStuckPages in apps/backend/features/source/repo.go), the orchestration layer is missing.
• Evidence: There is no ticker, cron, or background worker initialized in apps/backend/main.go to actually invoke this cleanup logic. The system currently possesses the ability to recover but lacks the instruction to do so, leaving the "Stuck Job Recovery" requirement unfulfilled.
4. Silent Operations in the Job Service
The Technical Constitution requires that every public operation must log its start, success, and failure using structured logging (slog).
• The Inconsistency: While the job feature's handler has been updated with logging, the job feature's service (apps/backend/features/job/service.go) remains entirely silent.
• Evidence: The Retry method performs I/O (database retrieval and NSQ publishing) but contains zero slog statements. If a retry fails or times out, there is no log trace within the service layer to diagnose the event, violating the "No Silent Failures" and "Structured Logging Only" rules.
5. Standard Library Logging Leak in Python Worker
The Constitution mandates the exclusive use of structured logging (e.g., structlog) and prohibits standard library string formatting.
• The Inconsistency: While the worker uses structlog for application events, it still relies on the standard pynsq library and tornado components which emit their own non-structured logs to the same output.
• Evidence: In apps/ingestion-worker/main.py, the code attempts to configure structured logging but the environment still captures raw traceback strings and tornado.iostream.StreamClosedError messages in a non-JSON format. This creates a "split-brain" logging environment where machine-parsing of logs will fail on critical infrastructure errors.
• The Best Solution: Instead of changing the libraries, you should configure the Python logging standard library to use a structlog processor as its output handler. This allows the worker to capture logs from internal libraries and format them into the same structured JSON used by your application code.

--------------------------------------------------------------------------------
Analogy: The codebase is like a shipping warehouse where the new sorting machines (Source/Settings) use barcodes and automated logs. However, the returns department (Job feature) still uses handwritten notes, and the delivery trucks (MCP SSE) occasionally forget the tracking numbers mid-route. Furthermore, while there is a policy to clear blocked aisles (Janitor logic), no one has been hired to actually walk the floor and perform the task
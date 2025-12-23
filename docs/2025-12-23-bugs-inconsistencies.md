Based on the current state of the codebase within the apps/ folder as represented in the sources, several inconsistencies remain despite recent stabilization efforts. These primarily involve deviations from the Technical Constitution and internal architectural misalignments.
1. API Response Envelope Inconsistency
The Technical Constitution (API Design) mandates a standard envelope format for all responses, using data and meta fields.
• The Inconsistency: While the error handlers in apps/backend/internal/settings/handler.go and apps/backend/features/source/handler.go have been updated to return a JSON envelope, the success paths still return raw objects.
• Example: GetSettings in the settings handler encodes the raw Settings struct directly, and the source List handler encodes a slice of Source objects without the required data or meta wrapping.
2. Correlation ID Generation vs. Propagation
A "Critical Constraint" of the project is that correlationId must be generated at ingress and propagated through the system.
• The Inconsistency: The current implementations in the backend handlers generate a new UUID inside the writeError helper function every time an error occurs.
• The Problem: This violates the requirement for a single, traceable ID. If a request is logged at the start with one ID (or none) and then returns an error with a newly generated ID, the ability to trace that specific request through the logs is broken.
3. Inconsistent Implementation of Timeouts
The "Universal Resource Management Rules" require all I/O operations to have explicit timeouts.
• The Inconsistency: While the reranker client uses a 10s timeout and the Python worker handlers use 60s asyncio.wait_for wrappers, the Backend Result Consumer lacks per-operation timeouts.
• Example: In apps/backend/internal/worker/result_consumer.go, the loop that calls h.embedder.Embed and h.store.StoreChunk uses a background context without an explicit context.WithTimeout, meaning a hang in the embedding API or vector store could block the consumer indefinitely.
4. Frontend Component Standardization
The project emphasizes using a unified design system via shadcn-vue.
• The Inconsistency: In apps/frontend/src/features/sources/SourceForm.vue, the implementation uses the standardized Input and Button components for the URL and depth. However, the Exclusions field is implemented as a raw, unstyled HTML <textarea>. This creates a visual and structural inconsistency with the rest of the form which uses the @/components/ui abstractions.
5. Worker Handler Return Types
There is a functional inconsistency in the contract between different worker handlers and the main processing loop in apps/ingestion-worker/main.py.
• The Inconsistency: The handle_web_task returns a list of dictionaries (to support recursive crawling), while handle_file_task returns a single string.
• The Result: This forces the main.py dispatcher to use different logic to wrap the results for the result producer, increasing the risk of bugs when new handler types (like GitHub or Sitemap) are added.
6. Logging Detail Disparity
The Technical Constitution requires entry and exit logging for all public operations.
• The Inconsistency: The Settings handler (Source 433) and MCP handler (Source 507) have been updated with "request received" and "request completed" logs. However, the Source handler (Source 535-539) lacks these entry/exit logs for its List, Delete, ReSync, and Get methods, only logging on specific operation failures.
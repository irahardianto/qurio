# Proposal: Refactor MCP from SSE to Streamable HTTP

**Date:** 2026-01-21
**Status:** Proposed

## Context
The current Model Context Protocol (MCP) implementation in `apps/backend/features/mcp/handler.go` uses the standard Server-Sent Events (SSE) transport. This approach requires two separate HTTP connections:
1.  A `GET` request for the event stream (server-to-client).
2.  A `POST` request for sending messages (client-to-server).

## Problem Statement
The current SSE implementation introduces several architectural challenges:
*   **Stateful Complexity:** The server must maintain a `sessions` map to link the `POST` messages back to the correct `GET` event stream. This complicates session management and scaling (requires sticky sessions or a distributed store for multiple replicas).
*   **Context Propagation:** Context (e.g., correlation IDs, tracing spans) is difficult to propagate correctly because the request handling (POST) is decoupled from the response stream (GET).
*   **Connection Overhead:** Maintaining two connections per active client increases resource usage.

## Proposed Solution
Refactor the MCP transport to use **Streamable HTTP**. This involves moving to a single-connection transport where both JSON-RPC requests and responses are streamed over a single `POST` request using chunked transfer encoding.

### Architectural Implications
*   **Statelessness:** The server becomes stateless regarding MCP transport sessions. Each request is self-contained or part of a persistent streaming connection, removing the need for a server-side `sessions` map.
*   **Reliability:** Context propagation is significantly improved. A request's context remains valid for the duration of the stream, simplifying tracing and logging.
*   **Simplicity:** Reduces code complexity by removing session management logic and locking mechanisms.
*   **Compatibility:** This is a deviation from the standard MCP SSE spec. Clients strictly adhering to the two-connection SSE model (like standard Claude Desktop configuration) may need updates or a compatibility layer if they don't support this transport mode.

## detailed Changes

### 1. `apps/backend/features/mcp/handler.go`
*   **Remove Session Management:** Delete the `sessions` map, `sessionsLock`, and associated logic for creating/tracking sessions.
*   **Unified Handler:** Replace `HandleSSE` and `HandleMessage` with a single `ServeHTTP` (or specific handler method) that:
    *   Accepts a `POST` request.
    *   Reads JSON-RPC requests from the request body stream.
    *   Writes JSON-RPC responses to the response body stream using chunked encoding.
    *   Keeps the connection open for the duration of the interaction.

### 2. `apps/backend/internal/app/app.go`
*   **Route Updates:** Remove the split routes (`/mcp/sse`, `/mcp/messages`) and register a single endpoint (e.g., `/mcp`) to the new streaming handler.

### 3. Testing
*   **Unit Tests (`handler_test.go`):** Update tests to simulate a full read/write stream on a single connection instead of mocking separate session/message interactions.
*   **Integration Tests (`handler_integration_test.go`):** Rewrite integration tests to verify the single-connection streaming behavior and ensure correlation IDs are correctly preserved in the response stream.

### 4. Documentation
*   **`README.md`:** Update connection instructions to reflect the single endpoint usage.

## Migration Plan
1.  Create a new branch for the refactor.
2.  Modify `handler.go` to implement the streaming logic.
3.  Update `app.go` routes.
4.  Refactor tests to pass with the new implementation.
5.  Verify end-to-end functionality.
6.  Update documentation.

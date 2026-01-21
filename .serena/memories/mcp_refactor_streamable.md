# MCP Streamable HTTP Refactor (Jan 21, 2026)

## Changes
- Removed stateful SSE (`/mcp/sse`) and separate message endpoint (`/mcp/messages`).
- Implemented unified stateless POST endpoint (`/mcp`).
- **Transport:** Standard JSON-RPC 2.0 over HTTP (Single Request -> Single Response).
- Removed `sessions` map and locking, simplifying concurrency model.
- Note: Initial implementation attempted infinite-loop streaming, but was reverted to single-request mode to resolve client compatibility issues ("Unexpected non-whitespace character").

## Rationale
- Statelessness improves reliability and simplicity (Technical Constitution).
- Align with native `httpUrl` support in modern MCP clients.
- Easier to test and debug (no complex session state management).
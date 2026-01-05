
## MCP Server Reliability (2026-01-05)
- **SSE Transport:**
  - **Keep-Alive:** Implemented 15s ticker to send `: keepalive` comments to prevent connection timeouts.
  - **Buffer:** Increased message channel buffer size from 10 to 100 to handle bursty traffic without dropping messages.
- **Tools:**
  - `qurio_list_sources`: Returns simplified list (ID, Name, Type). Falls back to URL if Name is empty.
  - `qurio_list_pages`: Returns simplified list (ID, URL). Status field omitted for cleaner agent context.

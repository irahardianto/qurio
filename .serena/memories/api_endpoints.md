# API Endpoints

Base URL: `/api` (Proxied via Nginx to Backend :8081)

## Response Format
All success responses follow this envelope:
```json
{
  "data": { ... }, // Object or Array
  "meta": { ... }  // Optional metadata (e.g. { "count": 10 })
}
```

All error responses follow this envelope:
```json
{
  "status": "error",
  "error": {
    "code": "...",
    "message": "..."
  },
  "correlationId": "..."
}
```

## Sources
| Method | Endpoint | Description | Payload/Params |
| :--- | :--- | :--- | :--- |
| `GET` | `/sources` | List all active sources | - |
| `GET` | `/sources/{id}` | Get source details & chunks | - |
| `POST` | `/sources` | Create new web source | `{"url": "...", "max_depth": 0, "exclusions": []}` |
| `POST` | `/sources/upload` | Upload document source | `multipart/form-data` (`file`: binary, max 50MB) |
| `DELETE` | `/sources/{id}` | Soft delete source | - |
| `POST` | `/sources/{id}/resync` | Trigger re-ingestion | - |
| `GET` | `/sources/{id}/pages` | List pages in source | - |

## Settings
| Method | Endpoint | Description | Payload/Params |
| :--- | :--- | :--- | :--- |
| `GET` | `/settings` | Get current config | - |
| `PUT` | `/settings` | Update config | `{"gemini_api_key": "...", "rerank_provider": "...", "rerank_api_key": "...", "search_alpha": 0.5, "search_top_k": 20}` |

## MCP (Model Context Protocol)
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/mcp` | Unified JSON-RPC 2.0 Endpoint (Streamable HTTP) |

## Health
| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` \| `/health` \| Service health check \| - \|

## Jobs (Failures)
| Method | Endpoint | Description | Payload/Params |
| :--- | :--- | :--- | :--- |
| `GET` | `/jobs/failed` | List all failed ingestion jobs | - |
| `POST` | `/jobs/{id}/retry` | Retry a failed job | - |

## Stats
| Method | Endpoint | Description | Payload/Params |
| :--- | :--- | :--- | :--- |
| `GET` | `/stats` | Get system counts (sources, docs, failures) | - |
# MCP Tools Specification

## Overview
Qurio exposes its knowledge base via the Model Context Protocol (MCP).

## Tools

### `qurio_search`
**Description:** Search & Exploration tool. Performs a hybrid search (Keyword + Vector). Use this for specific questions, finding code snippets, or exploring topics across known sources.

**Arguments:**
- `query` (string): The search query.
- `alpha` (number, 0.0-1.0): Hybrid search balance.
    - 0.0: Keyword only (Error codes, IDs)
    - 0.3: Mostly Keyword (Function names)
    - 0.5: Hybrid (Default)
    - 1.0: Vector (Conceptual questions)
- `limit` (integer): Max results (Default: 10, Max: 50).
- `source_id` (string): Filter results by source ID.
- `filters` (object): Metadata filters (e.g. `type='code'`, `language='go'`).

**Output:**
- List of results with Title, Type, Language, SourceID, and Content.
- Includes explicit instruction: "Use qurio_read_page(url=\"...\") to read the full content of any result."

### `qurio_list_sources`
**Description:** Discovery tool. Lists all available documentation sets (sources) currently indexed.

**Arguments:** None.

**Output:**
- List of sources (ID, Name, Type).

### `qurio_list_pages`
**Description:** Navigation tool. Lists all individual pages/documents within a specific source.

**Arguments:**
- `source_id` (string): The ID of the source.

**Output:**
- List of pages (ID, URL).

### `qurio_read_page`
**Description:** Deep Reading / Full Context tool. Retrieves the *entire* content of a specific page or document by its URL.

**Arguments:**
- `url` (string): The URL to fetch content for.

**Output:**
- Full page content (Chunks combined).

## Context Propagation
All tools propagate `correlationId` from the request to internal services for traceability.

## Error Handling
Tools return standard JSON-RPC 2.0 errors for internal failures (e.g., database connection issues, search failures).
- **Code:** -32603 (Internal Error)
- **Message:** Human-readable error description.
- **Data:** (Optional) Additional context.

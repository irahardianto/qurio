# Future Feature: Metadata-Based Search Filtering

**Status:** Proposed / Research
**Target:** Post-MVP
**Context:** Improving precision for code-specific queries vs. prose documentation.

## 1. Problem Statement
Currently, Qurio treats all ingested content as generic text chunks. A user searching for "auth middleware" gets a mix of:
1.  **Conceptual prose:** "Authentication middleware is used to..."
2.  **Implementation code:** `func AuthMiddleware(next http.Handler) ...`
3.  **API References:** "POST /auth/login returns 200..."

While Hybrid Search (`alpha`) helps prioritize exact matches, it cannot explicitly filter by *content type*. An agent asking for "Give me the code for auth middleware" might still get 5 paragraphs of text before finding the code block.

## 2. Proposed Solution: Metadata Filtering
Enhance the ingestion pipeline to classify chunks and allow explicit filtering via the MCP `search` tool.

### 2.1 Data Model Changes
Add a `type` field to the Weaviate schema `DocumentChunk`:
```graphql
class DocumentChunk {
    text: text
    metadata: object {
        source: string
        type: string  // "prose", "code", "api_spec", "config"
        language: string // "go", "python", "yaml" (if type="code")
    }
}
```

### 2.2 Ingestion Logic
During the chunking phase (`chunker.go`), implement basic heuristics or use a lightweight classifier:
-   **Code:** Detects heavy use of indentation, brackets `{}`, or specific keywords (`func`, `class`, `import`).
-   **Prose:** Standard sentence structure, paragraphs.
-   **Config:** Key-value pairs, YAML/JSON structure.

### 2.3 Retrieval Logic (MCP)
Update the `search` tool to accept a `filter` argument:
```json
{
  "name": "search",
  "arguments": {
    "query": "auth middleware implementation",
    "filter": { "type": "code", "language": "go" }
  }
}
```

## 3. Comparison with Other Approaches

| Approach | Implementation | Pros | Cons |
| :--- | :--- | :--- | :--- |
| **Unified (Current)** | Everything in one index. | Simple, zero overhead. | No explicit control over content type. |
| **Separate Indices** | `CodeIndex` vs `DocsIndex`. | Hard separation, easy to query only code. | Complex retrieval logic (which index to query?), duplication of context. |
| **Metadata Filtering (Proposed)** | Single index, `where` filter. | Flexible, allows "Code OR Prose", keeps architecture simple. | Requires accurate classification during ingestion. |
| **Reranking Only** | Use LLM to re-rank based on intent. | No schema changes needed. | Slower (latency), non-deterministic. |

## 4. Why Metadata Filtering is Superior
1.  **Agent-Friendly:** It gives the agent explicit tools (`filter={"type": "code"}`) which are deterministic, unlike trying to prompt-engineer the search query.
2.  **Performance:** Weaviate (and other vector DBs) perform pre-filtering very efficiently. Filtering down to "only code chunks" *before* vector search improves speed and relevance.
3.  **Simplicity:** It maintains the "Single Source of Truth" architecture (one index) while adding the necessary granularity.

## 5. Implementation Roadmap
1.  **Schema Migration:** Add `type` and `language` properties to Weaviate schema.
2.  **Chunker Upgrade:** Implement `DetectContentType(text string)` in the chunking service.
    -   *MVP:* Regex heuristics.
    -   *Advanced:* Tree-sitter parsing for exact language detection.
3.  **API Update:** Expose `filter` object in MCP search tool.

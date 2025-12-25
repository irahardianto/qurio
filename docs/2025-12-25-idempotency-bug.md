# Bug Report: Re-sync Idempotency Failure

**Date:** 2025-12-25
**Status:** Open
**Severity:** High (Data Integrity)
**Component:** Backend / Vector Store (Weaviate)

## Description
When triggering a "Re-sync" for an existing source, the system fails to delete the previously ingested chunks before storing the new ones. This results in duplicated chunks in the Vector Database (Weaviate), doubling the chunk count with every re-sync.

This occurs despite the implementation of `DeleteChunksByURL` in the ingestion pipeline.

## Steps to Reproduce
1.  **Ingest a File:** Upload a file (e.g., `mcp-test.md`) via the UI.
2.  **Verify Initial State:** Go to the "Source Details" page and note the chunk count (e.g., 1).
3.  **Trigger Re-sync:** Return to the "Sources" list and click the "Re-sync" button for the same source.
4.  **Wait:** Wait for the status to return to `completed`.
5.  **Verify Final State:** Go to the "Source Details" page.
    *   **Expected:** Chunk count is still 1.
    *   **Actual:** Chunk count is 2 (Duplicates visible in the list).

## Technical Context

### The Implementation
The `ResultConsumer` calls `DeleteChunksByURL` before storing new chunks:

```go
// apps/backend/internal/worker/result_consumer.go
if payload.URL != "" {
    if err := h.store.DeleteChunksByURL(ctx, payload.SourceID, payload.URL); err != nil {
        slog.Error("failed to delete old chunks", ...)
    }
}
```

The Weaviate adapter implements the deletion using `ObjectsBatchDeleter`:

```go
// apps/backend/internal/adapter/weaviate/store.go
func (s *Store) DeleteChunksByURL(ctx context.Context, sourceID, url string) error {
    _, err := s.client.Batch().ObjectsBatchDeleter().
        WithClassName("DocumentChunk").
        WithOutput("minimal").
        WithWhere(filters.Where().
            WithOperator(filters.And).
            WithOperands([]*filters.WhereBuilder{
                filters.Where().
                    WithPath([]string{"sourceId"}).
                    WithOperator(filters.Equal).
                    WithValueString(sourceID),
                filters.Where().
                    WithPath([]string{"url"}).
                    WithOperator(filters.Equal).
                    WithValueString(url),
            })).
        Do(ctx)
    return err
}
```

### Root Cause Analysis (Hypothesis)
The likely cause is a **Schema/Filter Mismatch** in Weaviate.

1.  **Schema Definition:**
    In `apps/backend/internal/vector/schema.go`, properties are defined as `text`:
    ```go
    { Name: "sourceId", DataType: []string{"text"} },
    { Name: "url", DataType: []string{"text"} },
    ```

2.  **Tokenization Issue:**
    *   In Weaviate, `text` properties are **tokenized** by default.
    *   A UUID like `bba0c598-7c3f...` splits into tokens: `bba0c598`, `7c3f`, etc.
    *   The `Equal` operator in the filter with the *full* UUID string might fail to match the individual tokens stored in the inverted index.
    *   Similarly, URLs like `/var/lib/...` are tokenized by slashes and punctuation.

### Recommended Fixes

#### Option A: Schema Update (Best Practice)
Change the data type of `sourceId` and `url` to `string` (or use property-level tokenization settings) to ensure they are treated as exact keywords.
*   **Note:** Changing schema for existing classes in Weaviate usually requires re-indexing (deletion and recreation of the class).

#### Option B: Filter Adjustment
Verify if Weaviate's Go client supports `Like` or matching on `id` (the UUID of the object) if we can verify the chunks differently. But since we delete by Source ID, we need to match the property.

#### Option C: Verification
Add a "Count" check after deletion in the `ResultConsumer` to verify chunks are gone before proceeding, raising an error if they persist (Strong Consistency).

## Relevant Files
- `apps/backend/internal/adapter/weaviate/store.go`
- `apps/backend/internal/worker/result_consumer.go`
- `apps/backend/internal/vector/schema.go`
- `apps/e2e/tests/ingestion.spec.ts` (Contains the failing test case `re-ingestion should replace chunks`)

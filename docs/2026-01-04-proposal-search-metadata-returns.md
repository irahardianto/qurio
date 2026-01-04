# Proposal: Enhanced Search Result Metadata & Citations

**Date:** 2026-01-04
**Status:** Draft
**Priority:** Medium (UX Enhancement)

## 1. Executive Summary
This proposal outlines the requirements for exposing rich document metadata—specifically **Author**, **Creation Date**, and **Page Count**—through the Search API. While recent backend updates utilize this data for improved *retrieval* (via contextual embeddings), the current API response format does not return these fields to the client. Exposing them will enable high-fidelity citations, improve user trust, and lay the groundwork for advanced frontend filtering.

## 2. Problem Statement
Currently, the ingestion pipeline successfully extracts and stores rich metadata (Author, CreatedAt, PageCount) in the Vector Database (Weaviate). However, the `GET /api/search` endpoint filters this information out, returning only the raw content and basic source info.

**Impact:**
- **Opaque Results:** A user might search for "financial report" and get a correct match, but they cannot immediately verify if it's the *2023* or *2024* report, or who authored it, without opening the full source.
- **Missed Citation Opportunity:** The frontend cannot display "Page X of Y" or "Authored by [Name]", which are critical cues for knowledge-intensive tasks.

## 3. Business Value & User Experience

### 3.1 Trusted Citations
By returning `author` and `created_at`, the UI can render search results with authoritative context.
*   *Before:* "Revenue increased by 5%..." (Source: internal-doc.pdf)
*   *After:* "Revenue increased by 5%..." (Source: internal-doc.pdf • **John Doe** • **Dec 2025**)

### 3.2 Navigation Context
Returning `page_count` and `chunk_index` allows the UI to show relative position.
*   *Example:* "Match found on page 3 of 45."

### 3.3 Foundation for Faceted Search
Exposing these fields is the prerequisite for building frontend features like "Sort by Date" or "Filter by Author" in the future.

## 4. Technical Requirements

### 4.1 Backend (`apps/backend`)
1.  **Update Domain Models:**
    - Extend `retrieval.SearchResult` struct to include `Author` (string), `CreatedAt` (string/time), and `PageCount` (int).
2.  **Enhance Data Access Layer:**
    - Update `Store.Search` and `Store.GetChunks` methods in `internal/adapter/weaviate/store.go`.
    - Modify the GraphQL query builder to request `author`, `createdAt`, and `pageCount` fields from Weaviate.
3.  **API Response Transformation:**
    - Ensure the `SearchResult` JSON marshaling includes these new fields (using `omitempty` where appropriate).

### 4.2 Frontend (`apps/frontend` - *For Future Implementation*)
*   *Note: This proposal focuses on enabling the API. Frontend consumption is a separate downstream task.*
*   Update `SearchCard` component to display metadata badges.

## 5. Acceptance Criteria
1.  **API Response Verification:**
    - A search query against a document with known metadata must return a JSON response containing `author`, `created_at`, and `page_count`.
2.  **Backward Compatibility:**
    - Documents ingested *before* the metadata enhancement (missing these fields) must still return valid responses (fields can be null/empty).
3.  **Performance:**
    - The addition of three text/int fields to the payload should have negligible impact on search latency (<5ms overhead).

## 6. Implementation Plan (Draft)
- **Task 1:** Update `SearchResult` DTO in retrieval service.
- **Task 2:** Modify Weaviate Adapter `Search` query to include new fields.
- **Task 3:** Add integration test verifying metadata presence in search response.

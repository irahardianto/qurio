---
name: technical-constitution
description: Generates technical implementation plans and architectural strategies that enforce the Project Constitution.
---

# Implementation Plan: Capabilities Enhancement

**Status:** Draft
**Context:** Upgrade ingestion, retrieval, and agent tooling per `2026-01-02-prd.md`.

## 1. Requirements Analysis

### Scope
Full implementation of the "Qurio Capabilities Enhancement" PRD, covering ingestion worker updates, backend chunking logic, vector store schema/filtering, and MCP tool upgrades.

### Gap Analysis
- **Nouns:**
    - `Title`: Needs extraction (Worker) and storage (Weaviate).
    - `Type`: Needs classification (`prose`, `code`, `api`, `config`) and storage.
    - `Language`: Needs extraction from fences and storage.
    - `Filter`: New object in Search API and MCP tool.
    - `Page`: New concept for `qurio_fetch_page` (aggregation of chunks).
- **Verbs:**
    - `Extract`: Title and Metadata (Worker/Chunker).
    - `Classify`: Content type (Chunker).
    - `Filter`: Weaviate `where` clauses (Store).
    - `Fetch`: Retrieve all chunks by URL (Store/MCP).

### Exclusions
- **Frontend:** No tasks scheduled for frontend changes in this plan (PRD focuses on Agent/Backend).
- **Weaviate Schema Migration:** PRD states "No migration needed" (schema-less), but we will verify property addition.

## 2. Knowledge Enrichment

**Simulated RAG via Codebase Analysis:**
- **Pattern 1 (Crawler):** `apps/ingestion-worker/handlers/web.py` uses `crawl4ai`. Result object has `markdown` but need to verify `title` access.
- **Pattern 2 (Chunking):** `apps/backend/internal/text/chunker.go` uses `strings.Fields`. Needs complete replacement with a markdown-aware state machine or library.
- **Pattern 3 (MCP):** `apps/backend/features/mcp/handler.go` uses manual JSON-RPC handling. Tools are defined in `tools/list` handler.
- **Pattern 4 (Store):** `apps/backend/internal/adapter/weaviate/store.go` uses `weaviate-go-client`. `WithProperties` needs update.

## 3. Implementation Tasks

### Task 1: Ingestion Worker - Title Extraction

**Files:**
- Modify: `apps/ingestion-worker/handlers/web.py`
- Test: `apps/ingestion-worker/tests/test_handlers.py`

**Requirements:**
- **Acceptance Criteria**
    1. `handle_web_task` returns a dictionary containing a `title` key.
    2. The title is extracted from the crawled page (via `crawl4ai` result or regex fallback).
- **Functional Requirements**
    - FR-01: System MUST identify and extract the title.
- **Test Coverage**
    - [Unit] `test_handle_web_task` - specific assertion for `result[0]["title"]`.

**Step 1: Write failing test**
```python
# apps/ingestion-worker/tests/test_handlers.py
import pytest
from handlers.web import handle_web_task

@pytest.mark.asyncio
async def test_handle_web_task_returns_title(mocker):
    # Mock crawl4ai result
    mock_result = mocker.MagicMock()
    mock_result.success = True
    mock_result.url = "http://example.com"
    mock_result.markdown = "# My Page Title\nSome content"
    mock_result.links = {'internal': []}
    # Assuming crawl4ai result might have metadata or we parse it
    
    mocker.patch('handlers.web.AsyncWebCrawler.arun', return_value=mock_result)
    
    result = await handle_web_task("http://example.com")
    
    assert "title" in result[0]
    # We might expect "My Page Title" if we implement parsing, or empty if not found
```

**Step 2: Verify test fails**
Run: `pytest apps/ingestion-worker/tests/test_handlers.py`
Expected: FAIL (KeyError: 'title' or Assertion Error)

**Step 3: Write minimal implementation**
```python
# apps/ingestion-worker/handlers/web.py
# ... inside handle_web_task ...
            # Extract title (simplistic regex fallback if not in result)
            title = ""
            if result.markdown:
                match = re.search(r'^#\s+(.+)$', result.markdown, re.MULTILINE)
                if match:
                    title = match.group(1).strip()
            
            return [{
                "url": result.url,
                "title": title, # Added
                "content": result.markdown,
                "links": internal_links
            }]
```

**Step 4: Verify test passes**
Run: `pytest apps/ingestion-worker/tests/test_handlers.py`
Expected: PASS

---

### Task 2: Backend - Markdown Chunker & Type Detection

**Files:**
- Modify: `apps/backend/internal/text/chunker.go`
- Test: `apps/backend/internal/text/chunker_test.go`

**Requirements:**
- **Acceptance Criteria**
    1. Chunking respects Markdown headers (does not split middle of header).
    2. Code blocks (```) are preserved as single chunks (unless > limit).
    3. Output chunks include `Type` (`prose`, `code`, `api`, `config`) and `Language`.
- **Functional Requirements**
    - FR-02: Classify chunks.
    - FR-03: No split code blocks.
    - FR-04: Line-based split for large blocks.
    - FR-05: Extract language.
- **Test Coverage**
    - [Unit] `ChunkMarkdown()` - Table driven tests with mixed prose/code inputs.

**Step 1: Write failing test**
```go
// apps/backend/internal/text/chunker_test.go
package text

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestChunkMarkdown_CodeBlockPreservation(t *testing.T) {
	input := `
# Header
Some prose.

` + "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```" + `

More prose.
`
	chunks := ChunkMarkdown(input, 100, 0) // Small size, but code block should stay intact
	
	foundCode := false
	for _, c := range chunks {
		if c.Type == ChunkTypeCode {
			foundCode = true
			assert.Equal(t, "go", c.Language)
			assert.Contains(t, c.Content, "func main()")
		}
	}
	assert.True(t, foundCode, "Should detect code block")
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/text/... -v`
Expected: FAIL (Undefined function ChunkMarkdown)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/text/chunker.go
package text

import (
	"strings"
	"regexp"
)

type ChunkType string

const (
	ChunkTypeProse  ChunkType = "prose"
	ChunkTypeCode   ChunkType = "code"
	ChunkTypeAPI    ChunkType = "api"
	ChunkTypeConfig ChunkType = "config"
	ChunkTypeCmd    ChunkType = "cmd"
)

type ChunkResult struct {
	Content  string
	Type     ChunkType
	Language string
}

// ChunkMarkdown implements a state-machine based chunker for Markdown
// This is a simplified version for the plan example
func ChunkMarkdown(text string, maxTokens, overlap int) []ChunkResult {
	var results []ChunkResult
	// ... Implementation of splitting logic ...
    // For "minimal implementation", we can use a regex to split code blocks vs text
    
    // Regex for code fences
    re := regexp.MustCompile("(?s)```(\w+)?\\n(.*?)\\n```")
    
    lastIndex := 0
    matches := re.FindAllStringSubmatchIndex(text, -1)
    
    for _, match := range matches {
        // Prose before code
        if match[0] > lastIndex {
            prose := strings.TrimSpace(text[lastIndex:match[0]])
            if len(prose) > 0 {
                results = append(results, ChunkResult{Content: prose, Type: ChunkTypeProse})
            }
        }
        
        // Code block
        lang := text[match[2]:match[3]]
        content := text[match[4]:match[5]]
        
        cType := ChunkTypeCode
        if lang == "yaml" || lang == "json" {
            cType = ChunkTypeConfig
        }
        
        results = append(results, ChunkResult{
            Content: "```" + lang + "\n" + content + "\n```",
            Type: cType,
            Language: lang,
        })
        
        lastIndex = match[1]
    }
    
    // Remaining prose
    if lastIndex < len(text) {
        prose := strings.TrimSpace(text[lastIndex:])
         if len(prose) > 0 {
            results = append(results, ChunkResult{Content: prose, Type: ChunkTypeProse})
        }
    }
    
	return results
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/text/... -v`
Expected: PASS

---

### Task 3: Retrieval Service & Weaviate Store Updates

**Files:**
- Modify: `apps/backend/internal/retrieval/service.go`
- Modify: `apps/backend/internal/adapter/weaviate/store.go`
- Modify: `apps/backend/internal/worker/types.go` (Update Chunk struct)
- Test: `apps/backend/internal/retrieval/service_test.go`

**Requirements:**
- **Acceptance Criteria**
    1. `SearchOptions` supports `Filters` (Type, Language).
    2. Weaviate `Search` applies these filters.
    3. `GetChunksByURL` method implemented.
- **Functional Requirements**
    - FR-07: Store metadata.
    - FR-09/10: Search filtering.
    - FR-11: Fetch full page.
- **Test Coverage**
    - [Unit] `Service.Search` - Verify filters are passed to store.

**Step 1: Write failing test**
```go
// apps/backend/internal/retrieval/service_test.go
// Add test case for filtering
func TestSearch_WithFilters(t *testing.T) {
    // ... setup mock store ...
    opts := &SearchOptions{
        Filters: map[string]interface{}{
            "type": "code",
        },
    }
    _, err := service.Search(ctx, "query", opts)
    // Assert mock store.Search was called with filters
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/retrieval/... -v`
Expected: FAIL (Field Filters undefined)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/retrieval/service.go
type SearchOptions struct {
	Alpha   *float32
	Limit   *int
	Filters map[string]interface{} // Added
}

// Update Store interface
type VectorStore interface {
	Search(ctx context.Context, query string, vector []float32, alpha float32, limit int, filters map[string]interface{}) ([]SearchResult, error)
    GetChunksByURL(ctx context.Context, url string) ([]SearchResult, error) // Added for FR-11
}

// apps/backend/internal/adapter/weaviate/store.go
// Implement updated Search with filters and GetChunksByURL
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/retrieval/... -v`
Expected: PASS

---

### Task 4: MCP Tooling Upgrade

**Files:**
- Modify: `apps/backend/features/mcp/handler.go`
- Test: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
    1. Tool `search` renamed to `qurio_search`.
    2. Tool `qurio_fetch_page` available.
    3. Descriptions match PRD.
- **Functional Requirements**
    - FR-12: `qurio_` prefix.
    - FR-13: Usage strategies in description.
- **Test Coverage**
    - [Unit] `tools/list` returns correct tools.
    - [Unit] `tools/call` with `qurio_search` works.

**Step 1: Write failing test**
```go
// apps/backend/features/mcp/handler_test.go
func TestToolsList_ReturnsQurioTools(t *testing.T) {
    // ...
    res := handler.processRequest(ctx, listReq)
    // Assert tool names
    assert.Equal(t, "qurio_search", res.Result.Tools[0].Name)
    assert.Equal(t, "qurio_fetch_page", res.Result.Tools[1].Name)
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/... -v`
Expected: FAIL (Name mismatch)

**Step 3: Write minimal implementation**
```go
// apps/backend/features/mcp/handler.go
// Rename "search" -> "qurio_search"
// Update description
// Add "qurio_fetch_page" to list
// Handle "qurio_fetch_page" in tools/call
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/... -v`
Expected: PASS

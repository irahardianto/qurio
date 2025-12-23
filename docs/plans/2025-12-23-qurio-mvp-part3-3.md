---
name: technical-constitution
description: Fixes critical bugs in MVP Part 3.2 implementation (Backend Handler and Python Worker) and standardizes payload.
---

# Implementation Plan - MVP Part 3.3: Crawler Fixes & Recursion

**Ref:** `2025-12-23-qurio-mvp-part3-3`
**Feature:** Crawler Integration Fixes
**Status:** Planned

## 1. Scope
Fix critical bugs in the previous deployment where `max_depth` and `exclusions` were ignored by both the Backend Handler and the Python Worker. Implement actual recursive crawling logic using `crawl4ai`'s `BFSDeepCrawlStrategy`. Repair broken backend tests.

**Gap Analysis:**
- **Backend:** `Create` handler only decodes `url`, dropping config.
- **Worker:** `handle_web_task` ignores `depth` and `exclusions`.
- **Worker Config:** Missing `GEMINI_API_KEY` for LLM filtering.
- **Tests:** `source_test.go` is broken/outdated.

## 2. Requirements

### Functional
- **Backend:** `POST /api/v1/sources` must accept `max_depth` (int) and `exclusions` ([]string).
- **Backend:** Publish NSQ payload with `max_depth` (standardized key).
- **Worker:** Implement recursive crawling using `BFSDeepCrawlStrategy`.
- **Worker:** Apply exclusions using `URLPatternFilter(reverse=True)`.
- **Worker:** Apply Advanced `crawl4ai` configuration:
    - `cache_mode=CacheMode.ENABLED`
    - `excluded_tags=['nav', 'footer', 'aside', 'header']`
    - `exclude_external_links=False`
    - **Filters:**
        - `PruningContentFilter(threshold=0.30, min_word_threshold=5, threshold_type="fixed")`
        - `LLMContentFilter` with `gemini/gemini-3-flash-preview` and specific extraction instruction.

### Non-Functional
- **Protocol:** Standardize on `max_depth` (JSON key) across Stack.
- **Testing:** Restore `go test` health.

## 3. Tasks

### Task 1: Fix Backend Source Handler
**Files:**
- Modify: `apps/backend/features/source/handler.go`
- Modify: `apps/backend/features/source/source.go` (Service.Create)
- Test: `apps/backend/features/source/handler_test.go` (Add payload check)

**Requirements:**
- **Acceptance Criteria**
  1. `Handler.Create` decodes full JSON body.
  2. `Service.Create` publishes `max_depth` and `exclusions`.
  3. API returns 201 with full Source object.

- **Test Coverage**
  - [Integration] `TestCreateSource_FullPayload` verifies DB persistence and NSQ publish.

**Step 1: Write failing test**
```go
// apps/backend/features/source/handler_test.go
func TestCreate_FullPayload(t *testing.T) {
    // Post JSON with max_depth: 2, exclusions: ["/blog"]
    // Assert Service called with correct struct fields
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/source/...`

**Step 3: Implementation**
```go
// handler.go
type CreateRequest struct {
    URL        string   `json:"url"`
    MaxDepth   int      `json:"max_depth"`
    Exclusions []string `json:"exclusions"`
}
// Decode into CreateRequest
// Map to &Source{...}
```

### Task 2: Update Worker Config & Infrastructure
**Files:**
- Modify: `apps/ingestion-worker/config.py`
- Modify: `docker-compose.yml`
- Modify: `apps/ingestion-worker/requirements.txt`

**Requirements:**
- **Acceptance Criteria**
  1. `config.py` includes `gemini_api_key` (loaded from env).
  2. `docker-compose.yml` passes `GEMINI_API_KEY` to `ingestion-worker`.
  3. `requirements.txt` includes `crawl4ai[google]` (or necessary extras).

**Step 3: Implementation**
```python
# apps/ingestion-worker/config.py
class Settings(BaseSettings):
    # ... existing ...
    gemini_api_key: str = "" # Env: GEMINI_API_KEY
```

```yaml
# docker-compose.yml
ingestion-worker:
  environment:
    - GEMINI_API_KEY=${GEMINI_API_KEY}
```

### Task 3: Implement Recursive Worker
**Files:**
- Modify: `apps/ingestion-worker/handlers/web.py`
- Modify: `apps/ingestion-worker/main.py`

**Requirements:**
- **Acceptance Criteria**
  1. Use `BFSDeepCrawlStrategy` if `max_depth > 0`.
  2. Use `URLPatternFilter` with `reverse=True` for exclusions.
  3. Configure `PruningContentFilter` with `threshold=0.30`, `min_word_threshold=5`.
  4. Configure `LLMContentFilter` with `gemini/gemini-3-flash-preview` and FULL instruction.
  5. Return `List[dict]` containing url and content.

**Step 3: Implementation**
```python
# handlers/web.py
from crawl4ai.deep_crawling import BFSDeepCrawlStrategy
from crawl4ai.deep_crawling.filters import URLPatternFilter, FilterChain
from crawl4ai import CrawlerRunConfig, CacheMode, LLMConfig
from crawl4ai.content_filter_strategy import PruningContentFilter, LLMContentFilter
from config import settings as app_settings

INSTRUCTION = """
    Extract technical content from this software documentation page.
    
    KEEP:
    - All code examples with their comments
    - Function/method signatures and parameters
    - Configuration examples and syntax
    - Technical explanations and concepts
    - Error messages and troubleshooting steps
    - Links to related API documentation
    
    REMOVE:
    - Navigation menus and sidebars
    - Copyright and legal notices
    - Unrelated marketing content
    - "Edit this page" links
    - Cookie banners and consent forms
    
    PRESERVE:
    - Code block language annotations (```go, etc.)
    - Heading hierarchy for context
    - Inline code references
    - Numbered lists for sequential steps
"""

async def handle_web_task(url: str, max_depth: int = 0, exclusions: list = None) -> list[dict]:
    # ... filters setup ...
    llm_filter = LLMContentFilter(
        provider="gemini/gemini-3-flash-preview",
        api_token=app_settings.gemini_api_key,
        enable_caching=True,
        instruction=INSTRUCTION,
        chunk_token_threshold=8000
    )
    
    config = CrawlerRunConfig(
        cache_mode=CacheMode.ENABLED,
        excluded_tags=['nav', 'footer', 'aside', 'header'],
        exclude_external_links=False,
        # ... markdown generator with filters ...
        deep_crawl_strategy=BFSDeepCrawlStrategy(...)
    )
    
    # Return list of { "url": r.url, "content": r.markdown }
```

### Task 4: Repair Backend Tests
**Files:**
- Modify: `apps/backend/features/source/source_test.go`
- Delete: `apps/backend/internal/worker/ingest_test.go` (Obsolete)

**Requirements:**
- Fix `NewService` signatures in tests.
- Ensure `go test ./...` passes.

**Step 1: Run tests**
`go test ./apps/backend/...`

**Step 2: Fix compilation errors**
Update mocks and constructors.

**Step 3: Verify pass**
`go test ./apps/backend/...`

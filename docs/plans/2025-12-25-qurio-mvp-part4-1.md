---
name: technical-constitution
description: Implementation plan for MVP Part 4.1 (Configuration, Agentic RAG & Verification).
---

# Implementation Plan - MVP Part 4.1: Configuration & Verification

**Ref:** `2025-12-25-qurio-mvp-part4-1`
**Feature:** Search Configuration, Agentic RAG, E2E Testing
**Status:** Draft

## 1. Scope
Address configuration gaps and enable "Agentic RAG" capabilities where the AI can dynamically tune search parameters (`alpha`, `limit`) per query. Establish E2E tests for these critical flows.

**Gap Analysis:**
- **Agent Agency:** Currently, the MCP tool only accepts `query`. It cannot adjust for "exact error match" vs "conceptual search".
- **Configuration:** `search_alpha` and `search_top_k` are hardcoded.
- **Verification:** No E2E tests for Search/Settings.

## 2. Requirements

### Functional
- **System Defaults:** Users can configure global default `Search Alpha` and `Top K` in Settings.
- **Agent Overrides:** The MCP `search` tool MUST accept optional `alpha` and `limit` arguments.
- **Priority:** `Agent Argument` > `System Default`.
- **E2E Testing:** Verify settings persistence and MCP tool overrides.

### Non-Functional
- **Tool Description:** The MCP tool definition must clearly explain *when* to use high vs low alpha to the AI.

## 3. Tasks

### Task 1: Database Migration
**Files:**
- Create: `apps/backend/migrations/000008_add_search_settings.up.sql`

**Requirements:**
- Add `search_alpha` (float, default 0.5) to `settings` table.
- Add `search_top_k` (int, default 20) to `settings` table.

**Implementation:**
```sql
ALTER TABLE settings ADD COLUMN IF NOT EXISTS search_alpha REAL DEFAULT 0.5;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS search_top_k INTEGER DEFAULT 20;
```

### Task 2: Backend Settings Update
**Files:**
- Modify: `apps/backend/internal/settings/service.go`
- Modify: `apps/backend/internal/settings/repo.go`

**Requirements:**
- Update `Settings` struct and SQL queries to include `SearchAlpha`, `SearchTopK`.

**Step 1: Write failing test**
Update `service_test.go` to assert new fields.

**Step 3: Implementation**
```go
type Settings struct {
    // ...
    SearchAlpha float32 `json:"search_alpha"`
    SearchTopK  int     `json:"search_top_k"`
}
```

### Task 3: Backend Retrieval Update (Agentic RAG)
**Files:**
- Modify: `apps/backend/internal/retrieval/service.go`

**Requirements:**
- Update `Search` signature to accept options: `Search(ctx, query, opts *SearchOptions)`.
- Logic: If `opts.Alpha` is set, use it. Else use `settings.SearchAlpha`. Same for `Limit`.

**Step 1: Write failing test**
Update `service_test.go`. Call `Search` with explicit alpha options and assert it overrides the mocked setting.

**Step 3: Implementation**
```go
type SearchOptions struct {
    Alpha *float32
    Limit *int
}

func (s *Service) Search(ctx context.Context, query string, opts *SearchOptions) ... {
    cfg, _ := s.settings.Get(ctx)
    
    alpha := cfg.SearchAlpha
    if opts != nil && opts.Alpha != nil {
        alpha = *opts.Alpha
    }
    
    limit := cfg.SearchTopK
    if opts != nil && opts.Limit != nil {
        limit = *opts.Limit
    }

    // ... pass to store.Search ...
}
```

### Task 4: MCP Tool Definition Update
**Files:**
- Modify: `apps/backend/features/mcp/handler.go`

**Requirements:**
- Update `InputSchema` for "search" tool.
- Add optional `alpha` (number, 0.0-1.0).
- Add optional `limit` (int).
- **Docstring:** Embed the usage table and code lookup examples directly into the tool description to guide the agent.

**Step 1: Implementation**
```go
// handler.go
description := `Search documentation and knowledge base.

ARGUMENT GUIDE:

[Alpha: Hybrid Search Balance]
- 0.0 (Keyword): Use for Error Codes ("0x8004"), IDs ("550e8400"), or unique strings.
- 0.3 (Mostly Keyword): Use for specific function names ("handle_web_task") where exact match matters but context helps.
- 0.5 (Hybrid - Default): Safe bet for general queries like "database configuration".
- 1.0 (Vector): Use for conceptual "How do I..." questions (e.g. "stop server" matches "shutdown").

[Limit: Result Count]
- Default: 10
- Recommended: 5-15 (Prevent context bloat)
- Max: 50
`
// ... in properties ...
"alpha": map[string]interface{}{
    "type": "number",
    "description": "Hybrid search balance (0.0=Keyword, 1.0=Vector). See tool description for guide.",
},
```

### Task 5: Frontend Settings UI & Tooltips
**Files:**
- Modify: `apps/frontend/src/features/settings/Settings.vue`
- Modify: `apps/frontend/src/features/settings/settings.store.ts`
- Create: `apps/frontend/src/components/ui/tooltip/` (Scaffold shadcn tooltip if missing, or use simple help icon)

**Requirements:**
- **Labeling:** Use user-friendly labels.
    - `Alpha` -> "Search Balance" (Slider). Labels: "Exact Match (0.0)" <-> "Conceptual (1.0)".
    - `Top K` -> "Max Results" (Input).
- **Tooltip:** Add info icon for "Search Balance": "Adjusts importance of Keyword vs Vector search. 0.0 for Error IDs, 1.0 for 'How to' questions."
- **Tooltip:** Add info icon for "Max Results": "Maximum number of document chunks to retrieve per search. Recommended: 10-20."

**Step 1: Implementation**
Add fields to store state and UI template with updated labels and tooltips.
**Files:**
- Modify: `apps/frontend/src/features/settings/Settings.vue`
- Modify: `apps/frontend/src/features/settings/settings.store.ts`

**Requirements:**
- Add Slider for Alpha (0-1, step 0.1) and Input for Top K.

### Task 6: E2E Tests
**Files:**
- Create: `apps/e2e/tests/search.spec.ts`

**Requirements:**
- Verify default search works.
- Verify search with `alpha` override works (no 500 error).

**Step 1: Implementation**
```typescript
test('MCP Search accepts alpha override', async ({ request }) => {
  const response = await request.post('http://localhost:8081/mcp', {
    data: {
      jsonrpc: '2.0',
      id: 1,
      method: 'tools/call',
      params: { 
        name: 'search', 
        arguments: { query: 'test', alpha: 0.1, limit: 5 } 
      }
    }
  });
  expect(response.ok()).toBeTruthy();
});
```
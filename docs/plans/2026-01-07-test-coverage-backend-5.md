# Plan: Backend Test Coverage & Logic Fixes

## Context
This plan addresses test coverage gaps and logic inconsistencies identified in `docs/2026-01-07-bug-testcoverage-backend-5.md`. It covers the Stats, Job, Source, and MCP features in the backend.

## Tasks

### Task 1: Stats Unit Coverage (Error Paths)

**Files:**
- Modify: `apps/backend/features/stats/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `TestHandler_GetStats` must verify that `JobRepo.Count` errors result in a correct error JSON response.
  2. Test must use table-driven approach covering: `Success`, `JobRepo.Count Error`.

- **Test Coverage**
  - [Unit] `TestHandler_GetStats_Table`

**Step 1: Write failing test**
```go
// apps/backend/features/stats/handler_test.go
func TestHandler_GetStats_Table(t *testing.T) {
    tests := []struct {
        name       string
        mockSetup  func(*MockJobRepo)
        wantStatus int
        wantError  bool
    }{
        {
            name: "JobRepo Error",
            mockSetup: func(m *MockJobRepo) {
                m.On("Count", mock.Anything, mock.Anything).Return(0, errors.New("db error"))
            },
            wantStatus: http.StatusInternalServerError,
            wantError:  true,
        },
    }
    // ... runner logic ...
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/stats/handler_test.go`
Expected: FAIL (if logic doesn't handle error or returns wrong status)

**Step 3: Write minimal implementation**
(Existing implementation might already be correct, but test confirms it. If not, modify `handler.go` to check error from `Count` and call `writeError`.)

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/stats/handler_test.go`
Expected: PASS


### Task 2: Job Service Unit Coverage (Publish Timeout)

**Files:**
- Modify: `apps/backend/features/job/service_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Verify `Retry` returns an error if NSQ publishing takes longer than 5 seconds.

- **Test Coverage**
  - [Unit] `TestService_Retry_Timeout`

**Step 1: Write failing test**
```go
// apps/backend/features/job/service_test.go
type SlowMockPublisher struct {
    mock.Mock
}
func (m *SlowMockPublisher) Publish(topic string, body []byte) error {
    time.Sleep(6 * time.Second) // Simulate timeout
    return nil
}

func TestService_Retry_Timeout(t *testing.T) {
    // Setup service with SlowMockPublisher
    // Call Retry
    // Assert error contains "timeout"
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/job/service_test.go`
Expected: FAIL

**Step 3: Write minimal implementation**
(Ensure `service.go` has `case <-time.After(5 * time.Second): return errors.New("timeout")` in its select block)

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/job/service_test.go`
Expected: PASS


### Task 3: Job Integration Coverage (DB Constraints)

**Files:**
- Create: `apps/backend/features/job/repo_integration_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Verify `failed_jobs` records are deleted when parent `source` is deleted (Cascade).
  2. Verify `List` returns jobs ordered by `created_at DESC`.

- **Test Coverage**
  - [Integration] `TestJobRepo_CascadeDelete`
  - [Integration] `TestJobRepo_List_Ordering`

**Step 1: Write failing test**
```go
// apps/backend/features/job/repo_integration_test.go
func (s *JobRepoSuite) TestCascadeDelete() {
    // Create Source
    // Create Failed Job linked to Source
    // Delete Source
    // Assert Job count is 0
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/job/repo_integration_test.go`
Expected: FAIL (Compilation error first time, then logic check)

**Step 3: Write minimal implementation**
(This validates the schema `migrations/000009...up.sql` and `repo.go` query logic)

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/job/repo_integration_test.go`
Expected: PASS


### Task 4: Source Upload Integration

**Files:**
- Create: `apps/backend/features/source/handler_integration_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Verify `POST /sources/upload` persists file to disk (`/var/lib/qurio/uploads` in container).
  2. Verify correct file metadata is returned.

- **Test Coverage**
  - [Integration] `TestHandler_Upload_Success`

**Step 1: Write failing test**
```go
// apps/backend/features/source/handler_integration_test.go
func (s *SourceHandlerSuite) TestUpload() {
    // Create Multipart Request
    // Call Handler
    // Verify File exists on Disk (s.UploadDir)
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/source/handler_integration_test.go`
Expected: FAIL

**Step 3: Write minimal implementation**
(Validates `handler.go` implementation)

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/source/handler_integration_test.go`
Expected: PASS


### Task 5: Source DB Integration (Partial Index)

**Files:**
- Create: `apps/backend/features/source/repo_integration_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Allow creation of Source with same content hash if previous one is soft-deleted.
  2. Reject creation if active one exists.

- **Test Coverage**
  - [Integration] `TestRepo_UniqueIndex_SoftDelete`

**Step 1: Write failing test**
```go
// apps/backend/features/source/repo_integration_test.go
func (s *SourceRepoSuite) TestUniqueIndex() {
    // Create Source A
    // Soft Delete Source A
    // Create Source A (Same Hash) -> Should Succeed
    // Create Source A (Same Hash) -> Should Fail
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/source/repo_integration_test.go`
Expected: FAIL

**Step 3: Write minimal implementation**
(Validates `repo.go` and schema)

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/source/repo_integration_test.go`
Expected: PASS


### Task 6: Source Logic (Exclusions Regex)

**Files:**
- Modify: `apps/backend/features/source/source.go`
- Modify: `apps/backend/features/source/source_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `Service.Create` must validate `Exclusions` field contains valid regex patterns.
  2. Return user-friendly error if invalid.

- **Test Coverage**
  - [Unit] `TestService_Create_InvalidRegex`

**Step 1: Write failing test**
```go
// apps/backend/features/source/source_test.go
func TestService_Create_InvalidRegex(t *testing.T) {
    // Call Create with Exclusions: ["["]
    // Assert Error is "invalid regex"
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/source/source_test.go`
Expected: FAIL (Currently allows it)

**Step 3: Write minimal implementation**
```go
// apps/backend/features/source/source.go
for _, pattern := range source.Exclusions {
    if _, err := regexp.Compile(pattern); err != nil {
        return fmt.Errorf("invalid exclusion regex: %s", pattern)
    }
}
```

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/source/source_test.go`
Expected: PASS


### Task 7: MCP Unit Coverage (Table Expansion)

**Files:**
- Modify: `apps/backend/features/mcp/handler_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Cover `qurio_read_page` with empty URL.
  2. Cover `qurio_search` with invalid Alpha (<0 or >1).
  3. Cover Malformed Filters.

- **Test Coverage**
  - [Unit] `TestHandler_ProcessRequest_Table` (Add Cases)

**Step 1: Write failing test**
(Add cases to existing `tests` slice in `handler_test.go`)

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/mcp/handler_test.go`
Expected: FAIL (If handlers don't check these bounds)

**Step 3: Write minimal implementation**
(Modify `handler.go` validation logic if needed)

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/mcp/handler_test.go`
Expected: PASS


### Task 8: MCP Integration (SSE Correlation)

**Files:**
- Create: `apps/backend/features/mcp/handler_integration_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. SSE connection establishment returns a Session ID.
  2. Messages sent with that Session ID preserve the Correlation ID in logs/processing.

- **Test Coverage**
  - [Integration] `TestHandler_SSE_Correlation`

**Step 1: Write failing test**
```go
// apps/backend/features/mcp/handler_integration_test.go
func (s *MCPSuite) TestSSE_Correlation() {
    // Connect SSE
    // Get Session ID
    // Send Message
    // Check Logs/Response for Correlation ID
}
```

**Step 2: Verify test fails**
Run: `go test -v apps/backend/features/mcp/handler_integration_test.go`
Expected: FAIL

**Step 3: Write minimal implementation**
(Validate `handler.go` context handling)

**Step 4: Verify test passes**
Run: `go test -v apps/backend/features/mcp/handler_integration_test.go`
Expected: PASS

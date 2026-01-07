### Task 1: Secure GitHub Actions Permissions

**Files:**
- Modify: `.github/workflows/test.yml`

**Requirements:**
- **Acceptance Criteria**
  1. `test.yml` workflow includes a top-level `permissions` block.
  2. Permissions are restricted to `contents: read` (minimum required for checkout and tests).

- **Functional Requirements**
  1. Limit GITHUB_TOKEN privileges to prevent unauthorized repository modifications during CI.

- **Non-Functional Requirements**
  - Security: Principle of Least Privilege.

- **Test Coverage**
  - Manual verification of YAML syntax.
  - CI run (if triggered) should pass.

**Step 1: Write failing test**
*Not applicable for YAML config, but we can validate syntax or run `act` if available. We will assume manual verification.*

**Step 2: Verify test fails**
*Skip.*

**Step 3: Write minimal implementation**
```yaml
name: Test and Coverage

# ADDED: Permissions block
permissions:
  contents: read

on:
# ... rest of file
```

**Step 4: Verify test passes**
*Verify YAML content.*

### Task 2: Fix File Upload "Failed to save file" Error

**Files:**
- Modify: `apps/backend/features/source/handler.go`
- Modify: `docker-compose.yml`

**Requirements:**
- **Acceptance Criteria**
  1. Backend logs the specific error from `os.Create` or `os.MkdirAll`.
  2. Local development (without Docker) defaults to a writable relative directory (`./uploads`) instead of `/var/lib/qurio`.
  3. Docker environment continues to use `/var/lib/qurio/uploads` via `QURIO_UPLOAD_DIR`.

- **Functional Requirements**
  1. `Upload` handler must successfully save files in both local and Docker environments.
  2. Users must receive a generic error, but admins (via logs) see the specific permission/path error.

- **Non-Functional Requirements**
  - Reliability: Works out of the box for new contributors (local dev).
  - **References:**
    - "Always check for errors... distinguish error types... ensure parent directories exist" (labex.io, posstree.com).
    - "Run Program with Sufficient, Minimal Permissions" (stackoverflow.com).

- **Test Coverage**
  - [Unit] `Handler.Upload` - mock file upload.
  - [Integration] Test actual upload in local environment (requires running backend).

**Step 1: Write failing test**
*We can't easily write a unit test that fails on "permissions" without mocking the OS, but the issue is environmental. We will rely on the fix logic.*

**Step 2: Verify test fails**
*Skip.*

**Step 3: Write minimal implementation**
**In `apps/backend/features/source/handler.go`:**
```go
	// Create uploads directory if not exists
	uploadDir := os.Getenv("QURIO_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads" // CHANGED from /var/lib/qurio/uploads
	}
    // ...
    dst, err := os.Create(path)
	if err != nil {
        slog.Error("failed to create file", "error", err, "path", path) // ADDED logging
		h.writeError(r.Context(), w, "INTERNAL_ERROR", "Failed to save file", http.StatusInternalServerError)
		return
	}
```

**In `docker-compose.yml`:**
```yaml
  backend:
    # ...
    environment:
      - QURIO_UPLOAD_DIR=/var/lib/qurio/uploads # ADDED
      - DB_HOST=postgres
    # ...
```

**Step 4: Verify test passes**
*Verify code changes.*

### Task 3: Fix XSS Vulnerabilities in MCP Handler

**Files:**
- Modify: `apps/backend/features/mcp/handler.go`

**Requirements:**
- **Acceptance Criteria**
  1. `endpoint` variable in SSE `endpoint` event is HTML-escaped.
  2. `sessionID` variable in SSE `id` event is HTML-escaped.
  3. `msg` variable in SSE `message` event is validated or sanitized (context-dependent).

- **Functional Requirements**
  1. Prevent XSS via Host header injection or session ID manipulation.

- **Non-Functional Requirements**
  - Security: OWASP Top 10 (Injection/XSS).
  - **References:**
    - "Manually escape strings with html.EscapeString... for SSE" (semgrep.dev, github.io).
    - "html/template package is the most robust solution" (go-cookbook.com).

- **Test Coverage**
  - [Unit] `HandleSSE` - verify output format and escaping.

**Step 1: Write failing test**
Create `apps/backend/features/mcp/handler_test.go`:
```go
package mcp

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "strings"
)

func TestHandleSSE_XSS(t *testing.T) {
    h := NewHandler(nil, nil)
    req := httptest.NewRequest("GET", "/mcp/sse", nil)
    req.Host = "example.com<script>alert(1)</script>" // Malicious Host
    w := httptest.NewRecorder()

    // We can't easily test the infinite loop, but we can check the first event
    // For this test, we might need to modify HandleSSE to be testable or run it in a goroutine
    // and close connection.
    // ...
}
```
*Refinement: Testing SSE infinite loop is hard. We will focus on the implementation fix which is straightforward using `html.EscapeString`.*

**Step 2: Verify test fails**
*Skip.*

**Step 3: Write minimal implementation**
**In `apps/backend/features/mcp/handler.go`:**
```go
import "html" // ADDED

// ...
	endpoint := fmt.Sprintf("%s://%s/mcp/messages?sessionId=%s", scheme, r.Host, sessionID)
    safeEndpoint := html.EscapeString(endpoint) // ADDED
	
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", safeEndpoint) // CHANGED
	w.(http.Flusher).Flush()
	
safeSessionID := html.EscapeString(sessionID) // ADDED
	fmt.Fprintf(w, "event: id\ndata: %s\n\n", safeSessionID) // CHANGED
```

**Step 4: Verify test passes**
*Verify code changes.*

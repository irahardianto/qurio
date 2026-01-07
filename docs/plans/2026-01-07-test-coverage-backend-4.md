### Task 1: Refactor Bootstrap and Test Retry Logic

**Files:**
- Modify: `apps/backend/internal/app/bootstrap.go`
- Test: `apps/backend/internal/app/bootstrap_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `Bootstrap` function delegates schema check to a helper `ensureSchemaWithRetry`.
  2. `ensureSchemaWithRetry` retries 10 times on failure.
  3. Unit test confirms retry logic works (fails X times then succeeds, or fails completely).

- **Functional Requirements**
  1. No behavior change in production (still retries).
  2. Testability improved by allowing mock injection into the retry logic helper.

- **Non-Functional Requirements**
  None for this task.

- **Test Coverage**
  - [Unit] `ensureSchemaWithRetry` with `MockVectorStore`.

**Step 1: Write failing test**
```go
// apps/backend/internal/app/bootstrap_test.go
package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"qurio/apps/backend/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestEnsureSchemaWithRetry_Success(t *testing.T) {
	mockStore := &app.MockVectorStore{
		EnsureSchemaErr: nil,
	}
	err := app.EnsureSchemaWithRetry(context.Background(), mockStore, 1, 1*time.Millisecond)
	assert.NoError(t, err)
}

func TestEnsureSchemaWithRetry_RetryThenSuccess(t *testing.T) {
	// This test requires a mock that changes behavior. 
	// Since MockVectorStore is simple, we might need a stateful mock or just test the failure case first.
	// For simplicity, let's test failure first.
	// NOTE: To test retry properly, we need a mock that counts calls or changes return value.
}

type statefulMockStore struct {
	app.MockVectorStore
	callCount int
	failUntil int
}

func (m *statefulMockStore) EnsureSchema(ctx context.Context) error {
	m.callCount++
	if m.callCount <= m.failUntil {
		return errors.New("schema error")
	}
	return nil
}

func TestEnsureSchemaWithRetry_Retries(t *testing.T) {
	mock := &statefulMockStore{failUntil: 2}
	err := app.EnsureSchemaWithRetry(context.Background(), mock, 5, 1*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 3, mock.callCount)
}

func TestEnsureSchemaWithRetry_Fail(t *testing.T) {
	mockStore := &app.MockVectorStore{
		EnsureSchemaErr: errors.New("permanent error"),
	}
	err := app.EnsureSchemaWithRetry(context.Background(), mockStore, 3, 1*time.Millisecond)
	assert.Error(t, err)
}
```

**Step 2: Verify test fails**
Run: `go test -v qurio/apps/backend/internal/app`
Expected: FAIL (undefined `EnsureSchemaWithRetry`)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/app/bootstrap.go
// Export this for testing
func EnsureSchemaWithRetry(ctx context.Context, store VectorStore, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = store.EnsureSchema(ctx); err == nil {
			return nil
		}
		if i < attempts-1 { // Don't sleep after last attempt
			time.Sleep(delay)
		}
	}
	return err
}

// In Bootstrap function, replace the loop with:
if err := EnsureSchemaWithRetry(ctx, vecStore, 10, 2*time.Second); err != nil {
    return nil, fmt.Errorf("weaviate schema error: %w", err)
}
```

**Step 4: Verify test passes**
Run: `go test -v qurio/apps/backend/internal/app`
Expected: PASS


### Task 2: Bootstrap Integration Test (Migrations)

**Files:**
- Create: `apps/backend/internal/app/bootstrap_integration_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `Bootstrap` runs successfully against a real Postgres container.
  2. SQL Migrations are applied (tables exist).
  3. Returns valid `Dependencies` struct.

- **Functional Requirements**
  1. Verify `migrate.NewWithDatabaseInstance` works with the actual SQL files.

- **Non-Functional Requirements**
  1. Use `Testcontainers` (via `IntegrationSuite`).

- **Test Coverage**
  - [Integration] `Bootstrap` against real DB/Weaviate/NSQ.

**Step 1: Write failing test**
```go
// apps/backend/internal/app/bootstrap_integration_test.go
package app_test

import (
	"context"
	"testing"

	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrap_Integration(t *testing.T) {
	suite := testutils.NewIntegrationSuite(t)
	suite.Setup()
	defer suite.Teardown()

	// Extract config from suite containers
	// Note: IntegrationSuite creates its own clients, but Bootstrap needs connection strings.
	// We need to extract them from the containers.
	// This might require helper methods in IntegrationSuite or accessing fields.
	// Assuming IntegrationSuite has public Containers or we can get connection strings.
	
	// Actually, suite.DB is already connected. 
	// Bootstrap creates its OWN connection. We need to feed it the right host/port.
	
	// For this test, we might need to "hack" the config to point to the random ports exposed by Docker.
	// But `suite` methods like `ConnectionString` exist.
	
	// Simulating Config based on Suite
	// This part depends on how IntegrationSuite exposes container details. 
	// For now, assume we can get mapped ports.
	
	// Placeholder for getting mapped ports (assuming testcontainers methods available)
	// We will fill this in implementation.
	
	cfg := &config.Config{
		// ... populated from suite containers ...
	}

	deps, err := app.Bootstrap(context.Background(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, deps)
	assert.NotNil(t, deps.DB)
	
	// Verify migration: Check if 'sources' table exists
	var exists bool
	err = deps.DB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'sources')").Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)
}
```

**Step 2: Verify test fails**
Run: `go test -v qurio/apps/backend/internal/app`
Expected: FAIL (Compilation error or test failure)

**Step 3: Write minimal implementation**
```go
// Refine the test in apps/backend/internal/app/bootstrap_integration_test.go to correctly get ports
// This involves using s.pgContainer.MappedPort(...) etc.
// Since IntegrationSuite fields are unexported in the `testutils` package (lowercase pgContainer), 
// we might need to expose them or add a `GetConfig()` method to IntegrationSuite.
// ACTION: Modify apps/backend/internal/testutils/integration_suite.go to add GetConfig() helper.
```

**Step 4: Verify test passes**
Run: `go test -v qurio/apps/backend/internal/app`
Expected: PASS


### Task 3: App.New Unit Test

**Files:**
- Modify: `apps/backend/internal/app/app_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `app.New` initializes all services and registers all routes.
  2. Middleware (CORS, CorrelationID) is applied (implicit via route registration check).
  3. No network calls are made (pure unit test with mocks).

- **Test Coverage**
  - [Unit] `New` constructor.

**Step 1: Write failing test**
```go
// apps/backend/internal/app/app_test.go
package app_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"qurio/apps/backend/internal/app"
	"qurio/apps/backend/internal/config"
)

func TestNew_Success(t *testing.T) {
	// Arrange
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	vecStore := &app.MockVectorStore{}
	taskPub := &app.MockTaskPublisher{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{}

	// Act
	application, err := app.New(cfg, db, vecStore, taskPub, logger)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, application)
	assert.NotNil(t, application.Handler)

	// Verify Routes
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/sources"},
		{"POST", "/sources"},
		{"GET", "/settings"},
		{"GET", "/stats"},
		{"GET", "/mcp/sse"},
	}

	ts := httptest.NewServer(application.Handler)
	defer ts.Close()

	for _, rt := range routes {
		req, _ := http.NewRequest(rt.method, ts.URL+rt.path, nil)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		// Should NOT be 404. Might be 401, 500, 200 depending on mock, but 404 means route not found.
		assert.NotEqual(t, http.StatusNotFound, resp.StatusCode, "Route %s %s not found", rt.method, rt.path)
	}
}
```

**Step 2: Verify test fails**
Run: `go test -v qurio/apps/backend/internal/app`
Expected: FAIL (if New logic is broken) or PASS (if currently working, acts as regression test).

**Step 3: Write minimal implementation**
(Existing implementation in `app.go` should satisfy this. This is backfilling coverage.)

**Step 4: Verify test passes**
Run: `go test -v qurio/apps/backend/internal/app`
Expected: PASS


### Task 4: Main Smoke Test

**Files:**
- Modify: `apps/backend/main.go` (Extract `run` function)
- Create: `apps/backend/smoke_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. `main` logic extracted to `run`.
  2. Smoke test spins up containers, calls `run`, verifies `/health`.
  3. Ensures full wiring works.

**Step 1: Write failing test**
```go
// apps/backend/smoke_test.go
package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"qurio/apps/backend/internal/config"
	"qurio/apps/backend/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmoke_Startup(t *testing.T) {
	// 1. Start Infrastructure
	suite := testutils.NewIntegrationSuite(t)
	suite.Setup()
	defer suite.Teardown()

	// 2. Configure App to use Infrastructure
	// We need to extract config from suite.
	// Since we are in `package main` we can't easily access `testutils` internals if they aren't exposed.
	// We'll rely on a helper or just manually grab ports if possible, or pass explicit config.
	
	// Assume we add a helper suite.GetAppConfig() -> *config.Config
	cfg := suite.GetAppConfig()
	cfg.APIKey = "test-key" // Bypass auth if needed or configured

	// 3. Run App in Background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// We need to call the extracted `run` function.
	// It's in main.go, package main. This test is package main.
	// So we can call `run(ctx, cfg)`.
	
	go func() {
		err := run(ctx, cfg) // This function needs to be created in main.go
		if err != nil && err != context.Canceled {
			t.Errorf("app run failed: %v", err)
		}
	}()

	// 4. Wait for Health Check
	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:8081/health")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 500*time.Millisecond)
}
```

**Step 2: Verify test fails**
Run: `go test -v qurio/apps/backend`
Expected: FAIL (undefined `run`, undefined `GetAppConfig`)

**Step 3: Write minimal implementation**
1.  **Refactor `main.go`**:
    ```go
    func run(ctx context.Context, cfg *config.Config) error {
        // Move body of main() here, accepts ctx and cfg.
        // Returns error instead of os.Exit
    }
    func main() {
        // ... Load config ...
        // call run(ctx, cfg)
    }
    ```
2.  **Update `testutils`**: Add `GetAppConfig()`.

**Step 4: Verify test passes**
Run: `go test -v qurio/apps/backend`
Expected: PASS

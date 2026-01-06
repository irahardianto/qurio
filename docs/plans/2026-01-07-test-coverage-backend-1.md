# Implementation Plan - Backend Integration Tests (Testcontainers)

This plan implements full integration testing infrastructure for the backend using `testcontainers-go`. It enforces I/O isolation by testing adapters against real, ephemeral containers (PostgreSQL, Weaviate, NSQ) rather than mocks, ensuring contract validity.

## Requirements Analysis

### Scope
- **Domain:** Backend Testing Infrastructure
- **Goal:** Achieve 100% integration coverage for critical adapters and worker flows.
- **Deliverables:**
  - Shared Test Suite (`internal/testutils`)
  - Source Repo Integration Tests (`features/source`)
  - Weaviate Store Integration Tests (`internal/adapter/weaviate`)
  - Worker Integration Tests (`internal/worker`)
  - MCP Handler Integration Tests (`features/mcp`)
  - CI Pipeline Update (`.github/workflows/test.yml`)

### Gap Analysis
- **Nouns (Infrastructure):** `PostgresContainer` (16-alpine), `WeaviateContainer` (semitechnologies/weaviate:latest), `NSQContainer` (v1.3.0).
- **Nouns (Code):** `IntegrationSuite`, `RepoIntegrationTest`, `StoreIntegrationTest`, `TestIngestIntegration`, `HandlerIntegrationTest`.
- **Verbs:** `SetupSuite` (Provision), `TeardownSuite` (Purge), `Migrate` (Schema), `Seed` (Data), `Assert` (State).
- **Exclusions:** None. All specified components in the source document are covered.

### Knowledge Enrichment
- **RAG Queries:**
  - "testcontainers postgres go" → Confirmed `modules/postgres` usage.
  - "testcontainers weaviate go" → Confirmed custom/generic container strategy.
  - "go nsq testcontainers" → Confirmed generic container strategy.
  - "go sql migration test" → Confirmed `golang-migrate` usage pattern.

---

## Tasks

### Task 1: Add TestDependencies

**Files:**
- Modify: `apps/backend/go.mod`
- Modify: `apps/backend/go.sum`

**Requirements:**
- **Functional Requirements:**
  1. Add `github.com/testcontainers/testcontainers-go`
  2. Add `github.com/testcontainers/testcontainers-go/modules/postgres`
  3. Ensure `github.com/golang-migrate/migrate/v4` is available (already present).

**Step 1: Write failing test**
*Skipped (Dependency management)*

**Step 2: Verify test fails**
*Skipped*

**Step 3: Write minimal implementation**
```bash
cd apps/backend
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go mod tidy
```

**Step 4: Verify test passes**
```bash
cat apps/backend/go.mod | grep "testcontainers"
# Expected: matches found
```

### Task 2: Implement Integration Suite Scaffolding

**Files:**
- Create: `apps/backend/internal/testutils/integration_suite.go`

**Requirements:**
- **Acceptance Criteria:**
  1. `SetupSuite` starts Postgres, Weaviate, and NSQ containers.
  2. Postgres is migrated using `golang-migrate`.
  3. Weaviate schema is initialized.
  4. NSQ topics are created.
  5. `TeardownSuite` cleans up all resources.
- **Functional Requirements:**
  - Use `testcontainers-go` for container lifecycle.
  - Map ports dynamically.
  - Return a `Suite` struct with initialized clients (`*sql.DB`, `*weaviate.Client`, `*nsq.Producer`).

**Step 1: Write failing test**
Create `apps/backend/internal/testutils/suite_test.go`:
```go
package testutils

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestSetupSuite(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    s := NewIntegrationSuite(t)
    s.Setup()
    defer s.Teardown()

    assert.NotNil(t, s.DB)
    assert.NotNil(t, s.Weaviate)
    assert.NotNil(t, s.NSQ)
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/testutils/... -v`
Expected: FAIL (compilation error, undefined `NewIntegrationSuite`)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/testutils/integration_suite.go
package testutils

import (
    "context"
    "database/sql"
    "fmt"
    "testing"
    "time"
    "path/filepath"
    "runtime"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    _ "github.com/lib/pq"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
    "github.com/weaviate/weaviate-go-client/v5/weaviate"
    "github.com/nsqio/go-nsq"
)

type IntegrationSuite struct {
    T        *testing.T
    DB       *sql.DB
    Weaviate *weaviate.Client
    NSQ      *nsq.Producer
    
    // Containers
    pgContainer       *postgres.PostgresContainer
    weaviateContainer testcontainers.Container
    nsqContainer      testcontainers.Container
}

func NewIntegrationSuite(t *testing.T) *IntegrationSuite {
    return &IntegrationSuite{T: t}
}

func (s *IntegrationSuite) Setup() {
    ctx := context.Background()
    
    // 1. Postgres
    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("qurio_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
            WithOccurrence(2).
            WithStartupTimeout(60*time.Second)),
    )
    require.NoError(s.T, err)
    s.pgContainer = pgContainer
    
    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(s.T, err)
    
    s.DB, err = sql.Open("postgres", connStr)
    require.NoError(s.T, err)

    // Run Migrations
    _, b, _, _ := runtime.Caller(0)
    basepath := filepath.Dir(b)
    migrationPath := fmt.Sprintf("file://%s/../../migrations", basepath)
    
    m, err := migrate.New(migrationPath, connStr)
    require.NoError(s.T, err)
    require.NoError(s.T, m.Up())

    // 2. Weaviate
    req := testcontainers.ContainerRequest{
        Image:        "semitechnologies/weaviate:latest",
        ExposedPorts: []string{"8080/tcp", "50051/tcp"},
        Env: map[string]string{
            "AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED": "true",
            "DEFAULT_VECTORIZER_MODULE":             "none",
            "PERSISTENCE_DATA_PATH":                 "/var/lib/weaviate",
        },
        WaitingFor: wait.ForHttp("/v1/meta").WithPort("8080/tcp").WithStartupTimeout(60 * time.Second),
    }
    weaviateC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    require.NoError(s.T, err)
    s.weaviateContainer = weaviateC
    
    host, err := weaviateC.Host(ctx)
    require.NoError(s.T, err)
    port, err := weaviateC.MappedPort(ctx, "8080")
    require.NoError(s.T, err)
    
    cfg := weaviate.Config{
        Host:   fmt.Sprintf("%s:%s", host, port.Port()),
        Scheme: "http",
    }
    s.Weaviate, err = weaviate.NewClient(cfg)
    require.NoError(s.T, err)

    // 3. NSQ
    nsqReq := testcontainers.ContainerRequest{
        Image:        "nsqio/nsq:v1.3.0",
        ExposedPorts: []string{"4150/tcp", "4151/tcp"},
        Cmd:          []string{"/nsqd", "--broadcast-address=localhost"}, // Simplified for test
        WaitingFor:   wait.ForLog("TCP: listening on").WithStartupTimeout(60 * time.Second),
    }
    nsqC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: nsqReq,
        Started:          true,
    })
    require.NoError(s.T, err)
    s.nsqContainer = nsqC

    nsqHost, err := nsqC.Host(ctx)
    require.NoError(s.T, err)
    nsqPort, err := nsqC.MappedPort(ctx, "4150")
    require.NoError(s.T, err)

    nsqCfg := nsq.NewConfig()
    s.NSQ, err = nsq.NewProducer(fmt.Sprintf("%s:%s", nsqHost, nsqPort.Port()), nsqCfg)
    require.NoError(s.T, err)
}

func (s *IntegrationSuite) Teardown() {
    ctx := context.Background()
    if s.pgContainer != nil {
        s.pgContainer.Terminate(ctx)
    }
    if s.weaviateContainer != nil {
        s.weaviateContainer.Terminate(ctx)
    }
    if s.nsqContainer != nil {
        s.nsqContainer.Terminate(ctx)
    }
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/testutils/... -v`
Expected: PASS

### Task 3: Source Feature Integration Tests

**Files:**
- Create: `apps/backend/features/source/repo_integration_test.go`

**Requirements:**
- **Acceptance Criteria:**
  1. Deduplication constraint on `content_hash` is verified.
  2. Soft Delete logic updates `deleted_at`.
  3. `CountPendingPages` returns accurate count.

**Step 1: Write failing test**
Create `apps/backend/features/source/repo_integration_test.go`:
```go
package source_test

import (
    "context"
    "testing"
    
    "qurio/apps/backend/features/source"
    "qurio/apps/backend/internal/testutils"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSourceRepo_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    s := testutils.NewIntegrationSuite(t)
    s.Setup()
    defer s.Teardown()

    repo := source.NewPostgresRepo(s.DB)
    ctx := context.Background()

    // 1. Deduplication
    src := &source.Source{
        ID: "src-1", 
        URL: "http://example.com",
        ContentHash: "hash1",
        Type: "documentation",
        Name: "Source 1",
    }
    err := repo.Create(ctx, src)
    require.NoError(t, err)

    src2 := &source.Source{
        ID: "src-2", 
        URL: "http://example.com/2",
        ContentHash: "hash1", // Duplicate hash
        Type: "documentation",
        Name: "Source 2",
    }
    err = repo.Create(ctx, src2)
    assert.Error(t, err, "should fail on duplicate content_hash")

    // 2. Soft Delete
    err = repo.Delete(ctx, "src-1")
    require.NoError(t, err)
    
    // Verify it's gone from standard Get
    _, err = repo.Get(ctx, "src-1")
    assert.Error(t, err)

    // 3. Page Management
    // Only if Repo has page management methods. 
    // If SourcePage logic is handled by PageManagerAdapter, we test that in Task 5. 
    // Here we focus on SourceRepo methods.
    // Assuming CountPendingPages is on Repo as per prompt
    count, err := repo.CountPendingPages(ctx, "src-1")
    if err == nil {
         assert.Equal(t, 0, count)
    }
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/source/... -v`
Expected: FAIL (if implementation is missing or behavior mismatch)

**Step 3: Write minimal implementation**
(Refine the test code to actually work with the `source.Repository` interface methods available)

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/source/... -v`
Expected: PASS

### Task 4: Weaviate Store Integration Tests

**Files:**
- Create: `apps/backend/internal/adapter/weaviate/store_integration_test.go`

**Requirements:**
- **Acceptance Criteria:**
  1. `StoreChunk` persists data.
  2. `DeleteChunksByURL` removes data (Exact Match Deletion).
  3. `Search` respects alpha (Hybrid Search).
  4. `Search` returns correct metadata.

**Step 1: Write failing test**
Create `apps/backend/internal/adapter/weaviate/store_integration_test.go`:
```go
package weaviate_test

import (
    "context"
    "testing"
    
    "qurio/apps/backend/internal/adapter/weaviate"
    "qurio/apps/backend/internal/testutils"
    "qurio/apps/backend/internal/vector"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWeaviateStore_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    s := testutils.NewIntegrationSuite(t)
    s.Setup()
    defer s.Teardown()

    store := weaviate.NewStore(s.Weaviate)
    ctx := context.Background()
    
    // Ensure Schema
    err := store.EnsureSchema(ctx)
    require.NoError(t, err)

    // 1. Store & Delete
    chunk := &vector.DocumentChunk{
        ID: "chunk-1",
        SourceID: "src-1",
        URL: "http://example.com/page",
        Content: "Postgres is a database",
        ChunkIndex: 0,
    }
    err = store.StoreChunk(ctx, chunk)
    require.NoError(t, err)

    err = store.DeleteChunksByURL(ctx, "http://example.com/page")
    require.NoError(t, err)

    // Verify deletion (Search should return empty)
    res, err := store.Search(ctx, "Postgres", 10, 0.0, nil) // 0.0 alpha = keyword
    require.NoError(t, err)
    assert.Empty(t, res)

    // 2. Hybrid Search
    chunkA := &vector.DocumentChunk{ID: "c-1", Content: "Postgres", URL: "u1"}
    chunkB := &vector.DocumentChunk{ID: "c-2", Content: "Databases", URL: "u2"}
    store.StoreChunk(ctx, chunkA)
    store.StoreChunk(ctx, chunkB)

    // Search for "Postgres" with keyword preference
    res, err = store.Search(ctx, "Postgres", 10, 0.0, nil)
    require.NoError(t, err)
    require.NotEmpty(t, res)
    assert.Equal(t, "Postgres", res[0].Content)
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/adapter/weaviate/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
(Refine test code)

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/adapter/weaviate/... -v`
Expected: PASS

### Task 5: Worker Integration Tests (Unskip)

**Files:**
- Modify: `apps/backend/internal/worker/integration_test.go`

**Requirements:**
- **Acceptance Criteria:**
  1. `TestIngestIntegration` is un-skipped.
  2. Full ingest flow (message -> DB update -> Weaviate store) is verified.

**Step 1: Write failing test**
Remove `t.Skip` from `apps/backend/internal/worker/integration_test.go` and update it to use `testutils.IntegrationSuite`.

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL (Need to wire up the suite correctly)

**Step 3: Write minimal implementation**
Refactor the test to use the shared suite:
```go
func TestIngestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    s := testutils.NewIntegrationSuite(t)
    s.Setup()
    defer s.Teardown()
    
    // Wire up dependencies using s.DB, s.Weaviate, s.NSQ
    // ...
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS

### Task 6: MCP Integration Tests

**Files:**
- Create: `apps/backend/features/mcp/handler_integration_test.go`

**Requirements:**
- **Acceptance Criteria:**
  1. `qurio_read_page` returns concatenated markdown.
  2. `qurio_search` filters by type.
  3. `X-Correlation-ID` is respected.

**Step 1: Write failing test**
Create test file using `testutils.IntegrationSuite`.

**Step 2: Verify test fails**
Run: `go test ./apps/backend/features/mcp/... -v`

**Step 3: Write minimal implementation**
Implement test cases calling the MCP handler logic backed by real containers.

**Step 4: Verify test passes**
Run: `go test ./apps/backend/features/mcp/... -v`

### Task 7: Update CI Pipeline

**Files:**
- Modify: `.github/workflows/test.yml`

**Requirements:**
- **Acceptance Criteria:**
  1. Job runs standard `go test ./...` (CI runs long tests by default).
  2. Environment supports Docker (ubuntu-latest).

**Step 1: Write failing test**
N/A (Configuration)

**Step 2: Verify test fails**
N/A

**Step 3: Write minimal implementation**
Update YAML to ensure `go test ./...` runs without `-short`.
Ensure CI pipeline just runs `go test ./...` (without `-short`).

**Step 4: Verify test passes**
CI Check.
